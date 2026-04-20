<template>
  <div ref="page" tabindex="-1" class="p-page p-page-photos not-selectable" :class="$config.aclClasses('photos')">
    <p-photo-toolbar
      ref="toolbar"
      :context="context"
      :filter="filter"
      :static-filter="staticFilter"
      :settings="settings"
      :refresh="refresh"
      :update-filter="updateFilter"
      :update-query="updateQuery"
      :on-close="onClose"
      :embedded="embedded"
      :semantic-search="semanticSearch"
      :toggle-semantic-search="toggleSemanticSearch"
      class="p-page__navigation"
    />

    <div v-if="loading" class="p-page__loading">
      <p-loading></p-loading>
    </div>
    <div v-else-if="semanticSearch" class="p-page__content">
      <div v-if="semanticResults.length === 0" class="pa-3">
        <v-alert color="surface-variant" icon="mdi-image-search-outline" class="no-results" variant="outlined">
          <div class="font-weight-bold">{{ $gettext("No semantic results found") }}</div>
          <div class="mt-2">{{ $gettext("Try a different query or disable semantic search.") }}</div>
        </v-alert>
      </div>
      <div v-else class="v-row search-results semantic-results pa-2">
        <div
          v-for="result in semanticResults"
          :key="result.id"
          class="v-col-6 v-col-sm-4 v-col-md-3 v-col-lg-2 pa-1"
        >
          <div class="media result semantic-result">
            <div
              class="preview"
              :style="`background-image: url(${result.url}); background-size: cover; background-position: center; height: 160px; border-radius: 4px; position: relative;`"
            >
              <div class="preview__overlay"></div>
              <button
                :title="result._liked ? $gettext('Liked') : $gettext('Like')"
                style="position:absolute; top:6px; right:6px; background:rgba(0,0,0,0.45); border:none; border-radius:50%; width:32px; height:32px; cursor:pointer; display:flex; align-items:center; justify-content:center;"
                @click.stop="likeSemanticResult(result)"
              >
                <i
                  class="mdi"
                  :class="result._liked ? 'mdi-heart' : 'mdi-heart-outline'"
                  :style="result._liked ? 'color:#f44336;font-size:18px;' : 'color:#fff;font-size:18px;'"
                />
              </button>
            </div>
            <div class="meta pa-1" style="font-size:11px; opacity:0.8;">
              {{ $gettext("Score") }}: {{ (result.score * 100).toFixed(0) }}%
            </div>
          </div>
        </div>
      </div>
    </div>
    <div v-else class="p-page__content">
      <p-scroll :hide-panel="hideExpansionPanel" :load-more="loadMore" :load-disabled="scrollDisabled" :load-distance="scrollDistance" :loading="loading">
      </p-scroll>

      <p-photo-clipboard :context="context" :refresh="refresh"></p-photo-clipboard>

      <p-photo-view-mosaic
        v-if="settings.view === 'mosaic'"
        :context="context"
        :photos="results"
        :select-mode="selectMode"
        :filter="filter"
        :edit-photo="editPhoto"
        :open-photo="openPhoto"
        :is-shared-view="isShared"
      ></p-photo-view-mosaic>
      <p-photo-view-list
        v-else-if="settings.view === 'list'"
        :context="context"
        :photos="results"
        :select-mode="selectMode"
        :filter="filter"
        :open-photo="openPhoto"
        :edit-photo="editPhoto"
        :open-date="openDate"
        :open-location="openLocation"
        :is-shared-view="isShared"
      ></p-photo-view-list>
      <p-photo-view-cards
        v-else
        :context="context"
        :photos="results"
        :select-mode="selectMode"
        :filter="filter"
        :open-photo="openPhoto"
        :edit-photo="editPhoto"
        :open-date="openDate"
        :open-location="openLocation"
        :is-shared-view="isShared"
      ></p-photo-view-cards>
    </div>
  </div>
</template>

<script>
import { Photo } from "model/photo";
import Thumb from "model/thumb";
import $api from "common/api";
import * as contexts from "options/contexts";
import PPhotoToolbar from "component/photo/toolbar.vue";
import PPhotoClipboard from "component/photo/clipboard.vue";
import PPhotoViewCards from "component/photo/view/cards.vue";
import PPhotoViewMosaic from "component/photo/view/mosaic.vue";
import PPhotoViewList from "component/photo/view/list.vue";
import PLoading from "component/loading.vue";
import PScroll from "component/scroll.vue";
import { getAppStorage } from "common/storage";

const appStorage = getAppStorage();

export default {
  name: "PPagePhotos",
  components: {
    PPhotoToolbar,
    PPhotoClipboard,
    PPhotoViewCards,
    PPhotoViewMosaic,
    PPhotoViewList,
    PLoading,
    PScroll,
  },
  props: {
    staticFilter: {
      type: Object,
      default: () => {},
    },
    onClose: {
      type: Function,
      default: undefined,
    },
    embedded: {
      type: Boolean,
      default: false,
    },
  },
  expose: ["onShortCut"],
  data() {
    const query = this.$route.query;
    const routeName = this.$route.name;
    const camera = query["camera"] ? parseInt(query["camera"]) : 0;
    const q = query["q"] ? query["q"] : "";
    const country = query["country"] ? query["country"] : "";
    const lens = query["lens"] ? parseInt(query["lens"]) : 0;
    const year = query["year"] ? parseInt(query["year"]) : 0;
    const month = query["month"] ? parseInt(query["month"]) : 0;
    const color = query["color"] ? query["color"] : "";
    const label = query["label"] ? query["label"] : "";
    const latlng = query["latlng"] ? query["latlng"] : "";
    const view = this.getViewType();
    const order = this.sortOrder();
    const filter = {
      country: country,
      camera: camera,
      lens: lens,
      label: label,
      latlng: latlng,
      year: year,
      month: month,
      color: color,
      order: order,
      reverse: this.sortReverse(),
      q: q,
    };

    const settings = this.$config.getSettings();
    const features = settings.features;

    if (settings) {
      if (features.private) {
        filter.public = "true";
      }

      if (features.review && (!this.staticFilter || !("quality" in this.staticFilter))) {
        filter.quality = "3";
      }
    }

    const batchSize = Photo.batchSize();

    return {
      isShared: this.$config.deny("photos", "manage"),
      canEdit: this.$config.allow("photos", "update") && features.edit,
      hasPlaces: this.$config.allow("places", "view") && features.places,
      canSearchPlaces: this.$config.allow("places", "search") && features.places,
      semanticSearch: false,
      semanticResults: [],
      subscriptions: [],
      listen: false,
      dirty: false,
      complete: false,
      results: [],
      scrollDisabled: true,
      scrollDistance: window.innerHeight * 4,
      batchSize: batchSize,
      offset: 0,
      page: 0,
      selection: this.$clipboard.selection,
      settings: {
        view,
      },
      filter: filter,
      lastFilter: {},
      routeName: routeName,
      loading: true,
      lightbox: {
        results: [],
        loading: false,
        complete: false,
        open: false,
        dirty: false,
        batchSize: batchSize * 4,
      },
    };
  },
  computed: {
    selectMode: function () {
      return this.selection.length > 0;
    },
    context: function () {
      return this.getContext();
    },
  },
  watch: {
    $route() {
      if (!this.$view.isActive(this)) {
        return;
      }

      this.$view.focus(this.$refs?.page);

      const query = this.$route.query;
      const settings = this.$config.getSettings();

      if (settings.features) {
        if (settings.features.private) {
          this.filter.public = "true";
        }

        if (settings.features.review && (!this.staticFilter || !("quality" in this.staticFilter))) {
          this.filter.quality = "3";
        }
      }

      this.filter.q = query["q"] ? query["q"] : "";
      // Disable semantic search when the text query is cleared.
      if (!this.filter.q) {
        this.semanticSearch = false;
      }
      this.filter.camera = query["camera"] ? parseInt(query["camera"]) : 0;
      this.filter.country = query["country"] ? query["country"] : "";
      this.filter.lens = query["lens"] ? parseInt(query["lens"]) : 0;
      this.filter.year = query["year"] ? parseInt(query["year"]) : 0;
      this.filter.month = query["month"] ? parseInt(query["month"]) : 0;
      this.filter.color = query["color"] ? query["color"] : "";
      this.filter.label = query["label"] ? query["label"] : "";
      this.filter.latlng = query["latlng"] ? query["latlng"] : "";
      this.filter.order = this.sortOrder();
      this.filter.reverse = this.sortReverse();

      this.settings.view = this.getViewType();

      /**
       * Even if the filter is unchanged, if the route is changed (for example
       * from `/review` to `/browse`), then the lastFilter must be reset, so that
       * a new search is actually triggered. That is because both routes use
       * this component, so it is reused by vue. See
       * https://github.com/photoprism/photoprism/pull/2782#issuecomment-1279821448.
       *
       * However, if the route is unchanged, the not resetting lastFilter prevents
       * unnecessary search-api-calls! These search-calls would otherwise reset
       * the view, even if we for example just returned from a fullscreen-download
       * in the ios-pwa. See
       * https://github.com/photoprism/photoprism/pull/2782#issue-1409954466
       */
      const routeChanged = this.routeName !== this.$route.name;

      if (routeChanged) {
        this.lastFilter = {};
        this.semanticSearch = false;
      }

      this.routeName = this.$route.name;

      this.search();
    },
  },
  created() {
    this.search();

    this.subscriptions.push(this.$event.subscribe("import.completed", (ev, data) => this.onImportCompleted(ev, data)));
    this.subscriptions.push(this.$event.subscribe("photos", (ev, data) => this.onUpdate(ev, data)));

    this.subscriptions.push(
      this.$event.subscribe("lightbox.opened", (ev, data) => {
        this.lightbox.open = true;
      })
    );
    this.subscriptions.push(
      this.$event.subscribe("lightbox.closed", (ev, data) => {
        this.lightbox.open = false;
      })
    );

    this.subscriptions.push(this.$event.subscribe("touchmove.top", () => this.refresh()));
    this.subscriptions.push(this.$event.subscribe("touchmove.bottom", () => this.loadMore()));
  },
  mounted() {
    this.$view.enter(this, this.$refs?.page);
  },
  beforeUnmount() {
    for (let i = 0; i < this.subscriptions.length; i++) {
      this.$event.unsubscribe(this.subscriptions[i]);
    }
  },
  unmounted() {
    this.$view.leave(this);
  },
  methods: {
    onShortCut(ev) {
      switch (ev.code) {
        case "Escape":
          if (document.activeElement instanceof HTMLInputElement) {
            document.activeElement.blur();
          }
          this.hideExpansionPanel();
          return true;
        case "KeyR":
          this.refresh();
          return true;
        case "KeyF":
          if (ev.shiftKey) {
            this.showExpansionPanel();
          }
          this.$view.focus(this.$refs?.toolbar, ".input-search input", true);
          return true;
        case "KeyU":
          if (this.$config.allow("files", "upload") && this.$config.feature("upload")) {
            this.$event.publish("dialog.upload");
          }
          return true;
      }
    },
    showExpansionPanel() {
      return this.$refs?.toolbar?.showExpansionPanel();
    },
    hideExpansionPanel() {
      return this.$refs?.toolbar?.hideExpansionPanel();
    },
    likeSemanticResult(result) {
      if (result._liked) return;
      result._liked = true;
      $api.post("semantic/like", { image_id: result.id, query: this.filter.q, score: result.score }).catch(() => {
        result._liked = false;
        this.$notify.warn(this.$gettext("Could not record like"));
      });
    },
    toggleSemanticSearch() {
      // Semantic mode requires a text query; reset it if the query was cleared.
      if (!this.filter.q) {
        this.semanticSearch = false;
        this.semanticResults = [];
        return;
      }
      this.semanticSearch = !this.semanticSearch;
      if (!this.semanticSearch) {
        this.semanticResults = [];
      }
      // Reset pagination and re-run search with the new mode.
      this.lastFilter = {};
      this.search();
    },
    searchCount() {
      const offset = parseInt(appStorage.getItem("photos.offset"));
      if (this.offset > 0 || !offset) {
        return this.batchSize;
      }
      return offset + this.batchSize;
    },
    setOffset(offset) {
      this.offset = offset;
      appStorage.setItem("photos.offset", offset);
    },
    getViewType() {
      if (this.embedded) {
        return "mosaic";
      }

      let queryParam = this.$route.query["view"] ? this.$route.query["view"] : "";
      let storedType = appStorage.getItem("photos.view");

      if (queryParam) {
        appStorage.setItem("photos.view", queryParam);
        return queryParam;
      } else if (storedType) {
        return storedType;
      } else if (window.innerWidth < 960) {
        return "mosaic";
      }

      return "cards";
    },
    getContext() {
      if (!this.staticFilter) {
        return contexts.Photos;
      }

      if (this.staticFilter.review) {
        return contexts.Review;
      } else if (this.staticFilter.archived) {
        return contexts.Archive;
      } else if (this.staticFilter.favorite) {
        return contexts.Favorites;
      } else if (this.staticFilter.hidden) {
        return contexts.Hidden;
      }

      return contexts.Default;
    },
    sortOrder() {
      if (this.embedded) {
        return "newest";
      }

      let storageKey;
      let defaultOrder;

      switch (this.getContext()) {
        case contexts.Archive:
          storageKey = "archive.order";
          defaultOrder = "archived";
          break;
        case contexts.Favorites:
          storageKey = "favorites.order";
          defaultOrder = "newest";
          break;
        case contexts.Hidden:
          storageKey = "hidden.order";
          defaultOrder = "added";
          break;
        case contexts.Review:
          storageKey = "review.order";
          defaultOrder = "added";
          break;
        default:
          storageKey = "photos.order";
          defaultOrder = "newest";
      }

      let queryOrder = this.$route.query["order"];
      let storageOrder = appStorage.getItem(storageKey);

      if (queryOrder) {
        appStorage.setItem(storageKey, queryOrder);
        return queryOrder;
      } else if (storageOrder) {
        return storageOrder;
      }

      return defaultOrder;
    },
    sortReverse() {
      return !!this.$route?.query["reverse"] && this.$route.query["reverse"] === "true";
    },
    openDate(index) {
      const photo = this.results[index];

      if (!photo) {
        return;
      } else if (!photo.TakenAt || photo.TakenAt.length < 10) {
        this.editPhoto(index);
        return;
      }

      const takenDate = photo.TakenAt.substring(0, 10);

      if (this.$isMobile) {
        this.$router.push({ query: { q: "taken:" + takenDate } });
      } else {
        const routeUrl = this.$router.resolve({ name: "all", query: { q: "taken:" + takenDate } }).href;
        if (routeUrl) {
          this.$util.openUrl(routeUrl);
        }
      }
    },
    openLocation(index) {
      if (!this.hasPlaces || !this.canSearchPlaces) {
        return;
      }

      const photo = this.results[index];

      if (!photo) {
        return;
      }

      if (photo.CellID && photo.CellID !== "zz") {
        this.$router.push({ name: "places", query: { q: photo.CellID } });
      } else if (photo.Country && photo.Country !== "zz") {
        this.$router.push({ name: "places", query: { q: "country:" + photo.Country } });
      } else {
        this.$notify.warn("unknown location");
      }
    },
    editPhoto(index, tab) {
      if (!this.canEdit) {
        return this.openPhoto(index);
      }

      let selection = this.results.map((p) => {
        return p.getId();
      });

      // Open Edit Dialog
      this.$event.publish("dialog.edit", { selection, album: null, index, tab });
    },
    openPhoto(index, showMerged = false) {
      if (this.loading || !this.listen || this.lightbox.loading || !this.results[index]) {
        return false;
      }

      const selected = this.results[index];

      // Do not open as stack if there is only one JPEG or if multiple pictures are selected.
      if (this.selection.length > 0 || selected.jpegFiles().length < 2) {
        showMerged = false;
      }

      if (showMerged) {
        this.$lightbox.openModels(Thumb.fromFiles([selected]), 0);
      } else if (this.filter?.order === "random") {
        this.$lightbox.openModels(Thumb.fromPhotos(this.results), index);
      } else {
        this.$lightbox.openView(this, index);
      }

      return true;
    },
    loadMore(force) {
      if (!force && (this.scrollDisabled || this.$view.isHidden(this))) {
        return;
      }

      this.scrollDisabled = true;
      this.listen = false;

      if (this.dirty) {
        this.lightbox.dirty = true;
      }

      const count = this.dirty ? (this.page + 2) * this.batchSize : this.batchSize;
      const offset = this.dirty ? 0 : this.offset;

      const params = {
        count: count,
        offset: offset,
        merged: true,
      };

      Object.assign(params, this.lastFilter);

      if (this.staticFilter) {
        Object.assign(params, this.staticFilter);
      }

      const loadFn = this.semanticSearch && this.lastFilter.q ? Photo.searchSemantic.bind(Photo) : Photo.search.bind(Photo);

      loadFn(params)
        .then((response) => {
          this.results = this.dirty ? response.models : Photo.mergeResponse(this.results, response);
          this.complete = response.count < response.limit;
          this.scrollDisabled = this.complete;

          if (this.complete) {
            this.setOffset(response.offset);
            if (!this.embedded && this.results.length > 1) {
              if (!this.lightbox.open) {
                this.$notify.info(this.$gettextInterpolate(this.$gettext("%{n} pictures found"), { n: this.results.length }));
              }
            }
          } else if (this.results.length >= Photo.limit()) {
            this.setOffset(response.offset);
            this.complete = true;
            this.scrollDisabled = true;
            if (!this.lightbox.open) {
              this.$notify.warn(this.$gettext("Can't load more, limit reached"));
            }
          } else {
            this.setOffset(response.offset + response.limit);
            this.offset = offset + count;
            this.page++;
            this.$nextTick(() => {
              if (this.$root.$el.clientHeight <= window.document.documentElement.clientHeight + 300) {
                this.loadMore();
              }
            });
          }
        })
        .catch(() => {
          this.scrollDisabled = false;
        })
        .finally(() => {
          this.dirty = false;
          this.loading = false;
          this.listen = true;
        });
    },
    updateSettings(props) {
      if (!props || typeof props !== "object" || props.target) {
        return;
      }

      for (const [key, value] of Object.entries(props)) {
        if (!this.settings.hasOwnProperty(key)) {
          continue;
        }
        switch (typeof value) {
          case "string":
            this.settings[key] = value.trim();
            break;
          default:
            this.settings[key] = value;
        }

        appStorage.setItem("photos." + key, this.settings[key]);
      }
    },
    updateFilter(props) {
      if (!props || typeof props !== "object" || props.target) {
        return;
      }

      for (const [key, value] of Object.entries(props)) {
        if (!this.filter.hasOwnProperty(key)) {
          continue;
        }
        switch (typeof value) {
          case "string":
            this.filter[key] = value.trim();
            break;
          default:
            this.filter[key] = value;
        }
      }
    },
    updateQuery(props) {
      this.updateFilter(props);

      if (this.loading) {
        return false;
      }

      const query = {
        view: this.settings.view,
      };

      Object.assign(query, this.filter);

      for (let key in query) {
        if (query[key] === undefined || !query[key]) {
          delete query[key];
        }
      }

      if (JSON.stringify(this.$route.query) === JSON.stringify(query)) {
        return false;
      }

      this.$router.replace({ query });

      return true;
    },
    searchParams() {
      const params = {
        count: this.searchCount(),
        offset: this.offset,
        merged: true,
      };

      Object.assign(params, this.filter);

      if (this.staticFilter) {
        Object.assign(params, this.staticFilter);
      }

      return params;
    },
    refresh(props) {
      this.updateSettings(props);

      // Do not refresh results if the view is already loading
      // or should not be listening for events.
      if (this.loading || !this.listen) {
        return;
      }

      // Make sure enough results are loaded to maintain the scroll position.
      if (this.page > 2) {
        this.page = this.page - 1;
      } else {
        this.page = 1;
      }

      // Flag results as dirty and incomplete to force a refresh.
      this.dirty = true;
      this.complete = false;

      // Enable infinite scrolling if it was disabled.
      this.scrollDisabled = false;

      this.loadMore(true);
    },
    reset() {
      this.results = [];
      this.lightbox.results = [];
    },
    search() {
      /**
       * search is called on mount or route change. If the route changed to an
       * open lightbox, no search is required. There is no reason to do an
       * initial results load, if the results aren't currently visible
       */
      if (this.lightbox.open) {
        return;
      }

      /**
       * re-creating the last scroll-position should only ever happen when using
       * back-navigation. We therefore reset the remembered scroll-position
       * in any other scenario
       */
      if (!this.$view.wasBackwardNavigation()) {
        this.setOffset(0);
      }
      this.scrollDisabled = true;

      // Don't query the same data more than once
      if (JSON.stringify(this.lastFilter) === JSON.stringify(this.filter)) {
        // this.$nextTick(() => this.$emit("scrollRefresh"));
        return;
      }

      Object.assign(this.lastFilter, this.filter);

      this.offset = 0;
      this.page = 0;
      this.loading = true;
      this.listen = false;
      this.complete = false;

      const params = this.searchParams();
      const searchFn = this.semanticSearch && this.filter.q ? Photo.searchSemantic.bind(Photo) : Photo.search.bind(Photo);

      searchFn(params)
        .then((response) => {
          // Hide search toolbar expansion panel when matching pictures were found.
          if (this.offset === 0 && response.count > 0) {
            this.hideExpansionPanel();
          }

          this.offset = response.limit;

          if (this.semanticSearch) {
            this.semanticResults = response.models;
            this.results = [];
          } else {
            this.semanticResults = [];
            this.results = response.models;
          }

          this.lightbox.results = [];
          this.lightbox.complete = false;
          this.complete = response.count < response.limit;
          this.scrollDisabled = this.complete;

          if (this.complete) {
            if (!this.results.length) {
              this.$notify.warn(this.$gettext("No pictures found"));
            } else if (!this.embedded && this.results.length === 1) {
              this.$notify.info(this.$gettext("One picture found"));
            } else if (!this.embedded) {
              this.$notify.info(this.$gettextInterpolate(this.$gettext("%{n} pictures found"), { n: this.results.length }));
            }
          } else {
            // this.$notify.info(this.$gettextInterpolate(this.$gettext("More than %{n} pictures found"), {n: 100}));
            this.$nextTick(() => {
              if (this.$root.$el.clientHeight <= window.document.documentElement.clientHeight + 300) {
                this.loadMore();
              }
            });
          }
        })
        .catch(() => {
          this.reset();
        })
        .finally(() => {
          this.dirty = false;
          this.loading = false;
          this.listen = true;
        });
    },
    onImportCompleted() {
      if (!this.listen) {
        return;
      }

      this.loadMore(true);
    },
    updateResults(entity) {
      this.results
        .filter((m) => m.UID === entity.UID)
        .forEach((m) => {
          for (let key in entity) {
            if (key !== "UID" && entity.hasOwnProperty(key) && entity[key] != null && typeof entity[key] !== "object") {
              m[key] = entity[key];
            }
          }
        });

      this.lightbox.results
        .filter((m) => m.UID === entity.UID)
        .forEach((m) => {
          for (let key in entity) {
            if (key !== "UID" && entity.hasOwnProperty(key) && entity[key] != null && typeof entity[key] !== "object") {
              m[key] = entity[key];
            }
          }
        });
    },
    removeResult(results, uid) {
      const index = results.findIndex((m) => m.UID === uid);

      if (index >= 0) {
        results.splice(index, 1);
      }
    },
    onUpdate(ev, data) {
      if (!this.listen) {
        return;
      }

      if (!data || !data.entities || !Array.isArray(data.entities)) {
        return;
      }

      const type = ev.split(".")[1];

      switch (type) {
        case "updated":
          for (let i = 0; i < data.entities.length; i++) {
            const values = data.entities[i];

            if (this.context === contexts.Review && values.Quality >= 3) {
              this.removeResult(this.results, values.UID);
              this.removeResult(this.lightbox.results, values.UID);
              this.$clipboard.removeId(values.UID);
            } else {
              this.updateResults(values);
            }
          }
          break;
        case "restored":
          this.dirty = true;
          this.complete = false;

          if (this.context !== contexts.Archive) break;

          for (let i = 0; i < data.entities.length; i++) {
            const uid = data.entities[i];

            this.removeResult(this.results, uid);
            this.removeResult(this.lightbox.results, uid);
          }

          break;
        case "archived":
          this.dirty = true;
          this.complete = false;

          if (this.context !== contexts.Archive) {
            for (let i = 0; i < data.entities.length; i++) {
              const uid = data.entities[i];

              this.removeResult(this.results, uid);
              this.removeResult(this.lightbox.results, uid);
              this.$clipboard.removeId(uid);
            }
          } else if (!this.results.length) {
            this.refresh();
          }

          break;
        case "deleted":
          this.dirty = true;
          this.complete = false;

          for (let i = 0; i < data.entities.length; i++) {
            const uid = data.entities[i];

            this.removeResult(this.results, uid);
            this.removeResult(this.lightbox.results, uid);
            this.$clipboard.removeId(uid);
          }

          break;
        case "created":
          this.dirty = true;
          this.scrollDisabled = false;
          this.complete = false;

          break;
        default:
          console.warn("unexpected event type", ev);
      }

      // TODO: Needed?
      this.$forceUpdate();
    },
  },
};
</script>

package search

// PhotosByUID returns PhotoResults for the given photo UIDs in the same order as uids.
func PhotosByUID(uids []string) (PhotoResults, error) {
	if len(uids) == 0 {
		return PhotoResults{}, nil
	}

	var rows PhotoResults

	if err := UnscopedDb().
		Table("files").
		Select(PhotosColsAll).
		Joins("JOIN photos ON files.photo_id = photos.id AND files.media_id IS NOT NULL").
		Joins("LEFT JOIN cameras ON photos.camera_id = cameras.id").
		Joins("LEFT JOIN lenses ON photos.lens_id = lenses.id").
		Joins("LEFT JOIN places ON photos.place_id = places.id").
		Where("photos.photo_uid IN (?) AND photos.deleted_at IS NULL AND files.file_primary = 1 AND files.file_error = ''", uids).
		Scan(&rows).Error; err != nil {
		return PhotoResults{}, err
	}

	// Re-order to match the caller-supplied UID ranking (insertion sort; n ≤ 100).
	rank := make(map[string]int, len(uids))
	for i, uid := range uids {
		rank[uid] = i
	}
	for i := 1; i < len(rows); i++ {
		for j := i; j > 0 && rank[rows[j].PhotoUID] < rank[rows[j-1].PhotoUID]; j-- {
			rows[j], rows[j-1] = rows[j-1], rows[j]
		}
	}

	return rows, nil
}

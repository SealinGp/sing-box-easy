package appupdate

type releaseView struct {
	Tag         string `json:"tag"`
	Name        string `json:"name"`
	Prerelease  bool   `json:"prerelease"`
	PublishedAt string `json:"published_at"`
	URL         string `json:"url"`
	Notes       string `json:"notes"`
	IsCurrent   bool   `json:"is_current"`
	IsNewer     bool   `json:"is_newer"`
}

func (u *Updater) ReleaseOverview(force bool) (any, error) {
	releases, err := u.Releases(force)
	if err != nil {
		return nil, err
	}

	current := Current()
	views := make([]releaseView, 0, len(releases))
	for _, r := range releases {
		view := releaseView{
			Tag:        r.TagName,
			Name:       r.Name,
			Prerelease: r.Prerelease,
			URL:        r.HTMLURL,
			Notes:      r.Body,
			IsCurrent:  CompareVersions(r.TagName, current) == 0 && IsKnown(),
			IsNewer:    IsNewer(r.TagName, current),
		}
		if !r.PublishedAt.IsZero() {
			view.PublishedAt = r.PublishedAt.UTC().Format("2006-01-02T15:04:05Z")
		}
		views = append(views, view)
	}

	return map[string]any{
		"current_version": current,
		"current_known":   IsKnown(),
		"releases":        views,
		"rate_limit":      u.RateLimit(),
	}, nil
}

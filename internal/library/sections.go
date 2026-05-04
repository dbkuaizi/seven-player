package library

func DefaultSections() []SectionInfo {
	return []SectionInfo{
		{
			ID:       "all",
			Label:    "全部",
			Icon:     "mdi-view-grid-outline",
			Color:    "#64748b",
			Children: []string{"全部"},
		},
		{
			ID:       "movies",
			Label:    "电影",
			Icon:     "mdi-movie-open-outline",
			Color:    "#d97706",
			Children: []string{"全部电影", "华语电影", "欧美电影", "日韩电影", "动画电影", "纪录电影"},
		},
		{
			ID:       "series",
			Label:    "剧集",
			Icon:     "mdi-television-classic",
			Color:    "#2563eb",
			Children: []string{"全部剧集", "国产剧", "英美剧", "日韩剧", "短剧"},
		},
		{
			ID:       "variety",
			Label:    "综艺",
			Icon:     "mdi-microphone-variant",
			Color:    "#ea580c",
			Children: []string{"全部综艺", "真人秀", "脱口秀", "音乐", "访谈", "竞技"},
		},
		{
			ID:       "anime",
			Label:    "动漫",
			Icon:     "mdi-animation-play-outline",
			Color:    "#7c3aed",
			Children: []string{"全部动漫", "番剧", "剧场版", "国漫"},
		},
		{
			ID:       "documentary",
			Label:    "纪录片",
			Icon:     "mdi-earth",
			Color:    "#059669",
			Children: []string{"全部纪录片", "自然", "历史", "科技", "人文"},
		},
	}
}

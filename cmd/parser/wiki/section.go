package wiki

// KnownRelease — конфигурация секции с играми с известной датой выхода.
var KnownRelease = SectionConfig{
	Label:             "Игры с известной датой выхода",
	SectionID:         "3",
	RemoveSubsections: []string{"8"},
	GameCol:           1,
	PlatformCol:       2,
}

// UnknownRelease — конфигурация секции с играми без даты выхода.
var UnknownRelease = SectionConfig{
	Label:             "Игры с неизвестной датой выхода",
	SectionID:         "8",
	RemoveSubsections: nil,
	GameCol:           0,
	PlatformCol:       1,
}

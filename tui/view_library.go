package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ryansantos40/go-music-player/utils"
)

func (m Model) renderLibraryColumn() string {
	var b strings.Builder

	b.WriteString(m.renderLibraryTabs())
	b.WriteString("\n\n")

	switch m.librarySection {
	case SectionPlaylists:
		b.WriteString(m.renderPlaylistsList())

	case SectionAlbums:
		b.WriteString(m.renderAlbumsList())

	case SectionArtists:
		b.WriteString(m.renderArtistsList())
	}

	return b.String()
}

func (m Model) renderLibraryTabs() string {
	labels := []struct {
		label   string
		section LibrarySection
	}{
		{"Playlists", SectionPlaylists},
		{"Albums", SectionAlbums},
		{"Artists", SectionArtists},
	}

	var parts []string
	for _, item := range labels {
		style := subtleStyle
		text := item.label
		if m.librarySection == item.section {
			style = selectedStyle
			text = fmt.Sprintf("[%s]", strings.ToUpper(item.label))
		}
		parts = append(parts, style.Render(text))
	}

	return strings.Join(parts, " ")
}

func (m Model) renderPlaylistsList() string {
	var b strings.Builder

	playlists := m.playlistStore.ListPlaylists()

	if len(playlists) == 0 {
		b.WriteString(subtleStyle.Render("No playlists yet"))
		b.WriteString("\n\n")
		b.WriteString(subtleStyle.Render("Press 'c' to create"))
		return b.String()
	}

	maxVisible := (m.height - 16) / 2
	start, end := clampWindow(m.playlistIndex, len(playlists), maxVisible)

	for i := start; i < end; i++ {
		name := playlists[i]
		trackCount := 0

		if playlist, err := m.playlistStore.GetPlaylist(name); err == nil {
			trackCount = len(playlist.Tracks)
		}

		line := fmt.Sprintf("%s (%d tracks)", name, trackCount)

		switch {
		case i == m.playlistIndex && m.focusedColumn == 0:
			b.WriteString(selectedStyle.Render("> " + line))
		case name == m.currentPlaylist:
			b.WriteString(statusStyle.Render("* " + line))
		default:
			b.WriteString(subtleStyle.Render("  " + line))
		}

		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderAlbumsList() string {
	var b strings.Builder
	albums := m.buildAlbumGroups()

	if len(albums) == 0 {
		b.WriteString(subtleStyle.Render("No albums yet"))
		return b.String()
	}

	maxVisible := (m.height - 16) / 2
	start, end := clampWindow(m.albumIndex, len(albums), maxVisible)

	for i := start; i < end; i++ {
		album := albums[i]
		line := fmt.Sprintf("%s - %s (%d)", album.Title, album.Artist, len(album.Tracks))

		if i == m.albumIndex && m.librarySection == SectionAlbums && m.focusedColumn == 0 {
			b.WriteString(selectedStyle.Render("> " + line))

		} else if m.currentFilter.Type == FilterAlbum && m.currentFilter.Key == album.Key {
			b.WriteString(statusStyle.Render("* " + line))

		} else {
			b.WriteString(subtleStyle.Render("  " + line))

		}

		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderArtistsList() string {
	var b strings.Builder
	artists := m.buildArtistGroups()

	if len(artists) == 0 {
		b.WriteString(subtleStyle.Render("No artists yet"))
		return b.String()
	}

	maxVisible := (m.height - 16) / 2
	start, end := clampWindow(m.artistIndex, len(artists), maxVisible)

	for i := start; i < end; i++ {
		artist := artists[i]
		line := fmt.Sprintf("%s (%d)", artist.Name, len(artist.Tracks))

		if i == m.artistIndex && m.librarySection == SectionArtists && m.focusedColumn == 0 {
			b.WriteString(selectedStyle.Render("> " + line))
		} else if m.currentFilter.Type == FilterArtist && m.currentFilter.Key == artist.Key {
			b.WriteString(statusStyle.Render("* " + line))
		} else {
			b.WriteString(subtleStyle.Render("  " + line))
		}

		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderTracksColumn() string {
	var b strings.Builder

	title := "--- [ ALL TRACKS ] ---"
	switch m.currentFilter.Type {
	case FilterPlaylist:
		title = fmt.Sprintf("--- [ PLAYLIST: %s ] ---", m.currentFilter.Label)
	case FilterAlbum:
		title = fmt.Sprintf("--- [ ALBUM: %s ] ---", m.currentFilter.Label)
	case FilterArtist:
		title = fmt.Sprintf("--- [ ARTIST: %s ] ---", m.currentFilter.Label)
	}

	b.WriteString(sectionTitleStyle.Width(m.width/3 - 4).Render(title))
	b.WriteString("\n\n")

	tracks := m.getFilteredTracks()

	if len(tracks) == 0 {
		b.WriteString(subtleStyle.Render("No tracks"))
		return b.String()
	}

	maxVisible := m.height - 16
	start := 0
	if m.selectedIndex >= maxVisible {
		start = m.selectedIndex - maxVisible + 1
	}

	end := start + maxVisible
	if end > len(tracks) {
		end = len(tracks)
	}

	for i := start; i < end; i++ {
		track := tracks[i]
		line := fmt.Sprintf("%d. %s - %s", i+1, track.Title, track.Artist)

		switch {
		case i == m.selectedIndex && m.focusedColumn == 1:
			b.WriteString(selectedStyle.Render("> " + line))
		case m.player != nil && track.Path == m.player.GetCurrentTrack().Path:
			b.WriteString(playingStyle.Render("♫ " + line))
		default:
			b.WriteString(subtleStyle.Render("  " + line))
		}

		b.WriteString("\n")
	}

	return b.String()
}

func clampWindow(index, total, maxVisible int) (int, int) {
	if maxVisible <= 0 {
		maxVisible = 1
	}

	if total <= maxVisible {
		return 0, total
	}

	if index < maxVisible {
		return 0, maxVisible
	}

	start := index - maxVisible + 1
	end := start + maxVisible

	if end > total {
		end = total
		start = end - maxVisible
	}

	return start, end
}

func (m *Model) syncLibrarySelection() {
	switch m.librarySection {
	case SectionPlaylists:
		m.applyPlaylistSelection()

	case SectionAlbums:
		m.applyAlbumSelection()

	case SectionArtists:
		m.applyArtistSelection()
	}
}

func (m *Model) navigateLibrary(delta int) {
	switch m.librarySection {
	case SectionPlaylists:
		playlists := m.playlistStore.ListPlaylists()
		if len(playlists) == 0 {
			return
		}
		m.playlistIndex = (m.playlistIndex + delta + len(playlists)) % len(playlists)
		m.applyPlaylistSelection()
	case SectionAlbums:
		albums := m.buildAlbumGroups()
		if len(albums) == 0 {
			return
		}
		m.albumIndex = (m.albumIndex + delta + len(albums)) % len(albums)
		m.applyAlbumSelection()
	case SectionArtists:
		artists := m.buildArtistGroups()
		if len(artists) == 0 {
			return
		}
		m.artistIndex = (m.artistIndex + delta + len(artists)) % len(artists)
		m.applyArtistSelection()
	}
}

func (m *Model) navigateTracks(delta int) {
	tracks := m.getFilteredTracks()
	if len(tracks) == 0 {
		return
	}
	m.selectedIndex = (m.selectedIndex + delta + len(tracks)) % len(tracks)
}

func (m *Model) handleTrackSelection() {
	tracks := m.getFilteredTracks()
	if len(tracks) == 0 || m.selectedIndex >= len(tracks) {
		return
	}

	switch m.currentFilter.Type {
	case FilterPlaylist:
		if playlist, err := m.playlistStore.GetPlaylist(m.currentFilter.Key); err == nil {
			m.player = utils.NewPlayer(playlist.Tracks)
			_ = m.player.Skip(m.selectedIndex)
		}
	case FilterAlbum, FilterArtist:
		m.player = utils.NewPlayer(tracks)
		_ = m.player.Skip(m.selectedIndex)
	default:
		if m.player == nil {
			m.player = utils.NewPlayer(m.tracks)
		}
		_ = m.player.Skip(m.selectedIndex)
	}

	m.lastTrackIdx = m.selectedIndex
}

func (m *Model) applyPlaylistSelection() {
	playlists := m.playlistStore.ListPlaylists()
	if len(playlists) == 0 {
		m.setFilterAll()
		return
	}

	if m.playlistIndex >= len(playlists) {
		m.playlistIndex = len(playlists) - 1
	}
	if m.playlistIndex < 0 {
		m.playlistIndex = 0
	}

	name := playlists[m.playlistIndex]
	m.currentPlaylist = name
	m.currentFilter = TrackFilter{
		Type:  FilterPlaylist,
		Key:   name,
		Label: name,
	}
	m.selectedIndex = 0
}

func (m *Model) applyAlbumSelection() {
	albums := m.buildAlbumGroups()
	if len(albums) == 0 {
		m.setFilterAll()
		return
	}

	if m.albumIndex >= len(albums) {
		m.albumIndex = len(albums) - 1
	}
	if m.albumIndex < 0 {
		m.albumIndex = 0
	}

	album := albums[m.albumIndex]
	m.currentPlaylist = ""
	m.currentFilter = TrackFilter{
		Type:  FilterAlbum,
		Key:   album.Key,
		Label: fmt.Sprintf("%s — %s", album.Title, album.Artist),
	}
	m.selectedIndex = 0
}

func (m *Model) applyArtistSelection() {
	artists := m.buildArtistGroups()
	if len(artists) == 0 {
		m.setFilterAll()
		return
	}

	if m.artistIndex >= len(artists) {
		m.artistIndex = len(artists) - 1
	}
	if m.artistIndex < 0 {
		m.artistIndex = 0
	}

	artist := artists[m.artistIndex]
	m.currentPlaylist = ""
	m.currentFilter = TrackFilter{
		Type:  FilterArtist,
		Key:   artist.Key,
		Label: artist.Name,
	}
	m.selectedIndex = 0
}

func (m *Model) setFilterAll() {
	m.currentPlaylist = ""
	m.currentFilter = TrackFilter{
		Type:  FilterAll,
		Key:   "",
		Label: "All Tracks",
	}
	m.selectedIndex = 0
}

func (m Model) getFilteredTracks() []utils.Track {
	switch m.currentFilter.Type {
	case FilterPlaylist:
		if playlist, err := m.playlistStore.GetPlaylist(m.currentFilter.Key); err == nil {
			return playlist.Tracks
		}
	case FilterAlbum:
		for _, album := range m.buildAlbumGroups() {
			if album.Key == m.currentFilter.Key {
				return album.Tracks
			}
		}
	case FilterArtist:
		for _, artist := range m.buildArtistGroups() {
			if artist.Key == m.currentFilter.Key {
				return artist.Tracks
			}
		}
	}
	return m.tracks
}

func (m Model) buildAlbumGroups() []AlbumGroup {
	groups := map[string]*AlbumGroup{}

	for _, track := range m.tracks {
		title := track.Album
		if title == "" {
			title = filepath.Base(filepath.Dir(track.Path))
		}
		if title == "" {
			title = "Unknown Album"
		}

		artist := track.Artist
		if artist == "" {
			artist = "Unknown Artist"
		}

		key := strings.ToLower(title + "|" + artist)
		group, ok := groups[key]
		if !ok {
			group = &AlbumGroup{
				Key:    key,
				Title:  title,
				Artist: artist,
				Year:   track.Year,
			}
			groups[key] = group
		}
		group.Tracks = append(group.Tracks, track)
	}

	albums := make([]AlbumGroup, 0, len(groups))
	for _, group := range groups {
		albums = append(albums, *group)
	}

	sort.Slice(albums, func(i, j int) bool {
		if albums[i].Title == albums[j].Title {
			return albums[i].Artist < albums[j].Artist
		}
		return albums[i].Title < albums[j].Title
	})

	return albums
}

func (m Model) buildArtistGroups() []ArtistGroup {
	groups := map[string]*ArtistGroup{}

	for _, track := range m.tracks {
		name := track.Artist
		if name == "" {
			name = "Unknown Artist"
		}
		key := strings.ToLower(name)

		group, ok := groups[key]
		if !ok {
			group = &ArtistGroup{
				Key:  key,
				Name: name,
			}
			groups[key] = group
		}
		group.Tracks = append(group.Tracks, track)
	}

	artists := make([]ArtistGroup, 0, len(groups))
	for _, group := range groups {
		artists = append(artists, *group)
	}

	sort.Slice(artists, func(i, j int) bool {
		return artists[i].Name < artists[j].Name
	})

	return artists
}

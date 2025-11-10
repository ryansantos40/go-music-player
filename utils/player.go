package utils

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/flac"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"
)

type RepeatMode int

const (
	RepeatOff RepeatMode = iota
	RepeatOne
	RepeatAll
)

func (r RepeatMode) String() string {
	switch r {
	case RepeatOff:
		return " off"
	case RepeatOne:
		return "🔂 one"
	case RepeatAll:
		return "🔁 all"
	default:
		return "off"
	}
}

type Player struct {
	tracks         []Track
	shuffledTracks []Track
	currentIndex   int
	playing        bool
	paused         bool
	shuffle        bool
	repeatMode     RepeatMode
	streamer       beep.StreamSeekCloser
	ctrl           *beep.Ctrl
	format         beep.Format
	currentTime    time.Duration
	totalTime      time.Duration
	speakerInit    bool
	mu             sync.Mutex
	currentFile    *os.File
	volume         float64
	volumeCtrl     *effects.Volume
}

func decodeAudioFile(f *os.File) (beep.StreamSeekCloser, beep.Format, error) {
	ext := strings.ToLower(filepath.Ext(f.Name()))
	switch ext {
	case ".mp3":
		return mp3.Decode(f)
	case ".wav":
		return wav.Decode(f)
	case ".flac":
		return flac.Decode(f)
	default:
		return nil, beep.Format{}, fmt.Errorf("unsupported audio format: %s", ext)
	}
}

func NewPlayer(tracks []Track) *Player {
	return &Player{
		tracks:         tracks,
		shuffledTracks: nil,
		currentIndex:   0,
		playing:        false,
		shuffle:        false,
		repeatMode:     RepeatOff,
		speakerInit:    false,
		volume:         1.0,
	}
}

func (p *Player) createShuffledPlaylist() {
	p.shuffledTracks = make([]Track, len(p.tracks))
	copy(p.shuffledTracks, p.tracks)

	rand.Seed(time.Now().UnixNano())
	for i := len(p.shuffledTracks) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		p.shuffledTracks[i], p.shuffledTracks[j] = p.shuffledTracks[j], p.shuffledTracks[i]
	}
}

func (p *Player) getCurrentPlaylist() []Track {
	if p.shuffle && p.shuffledTracks != nil {
		return p.shuffledTracks
	}
	return p.tracks
}

func (p *Player) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.ctrl != nil {
		speaker.Lock()
		p.ctrl.Paused = true
		p.ctrl.Streamer = nil
		speaker.Unlock()
	}

	speaker.Clear()

	if p.streamer != nil {
		p.streamer.Close()
		p.streamer = nil
	}

	if p.currentFile != nil {
		p.currentFile.Close()
		p.currentFile = nil
	}

	p.playing = false
	p.paused = false
	p.currentTime = 0
}

func (p *Player) Play() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	playlist := p.getCurrentPlaylist()
	if p.currentIndex >= len(playlist) {
		return nil
	}

	if p.streamer != nil {
		p.streamer.Close()
		p.streamer = nil
	}
	if p.currentFile != nil {
		p.currentFile.Close()
		p.currentFile = nil
	}

	track := playlist[p.currentIndex]
	f, err := os.Open(track.Path)
	if err != nil {
		return err
	}

	streamer, format, err := decodeAudioFile(f)
	if err != nil {
		f.Close()
		return err
	}

	p.currentFile = f
	p.streamer = streamer
	p.format = format
	p.totalTime = time.Duration(float64(streamer.Len()) / float64(format.SampleRate) * float64(time.Second))

	if !p.speakerInit {
		speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
		p.speakerInit = true
	}

	p.volumeCtrl = &effects.Volume{
		Streamer: p.streamer,
		Base:     2,
		Volume:   p.volumeToDecibels(p.volume),
		Silent:   p.volume < 0.0001,
	}

	done := make(chan bool)
	p.ctrl = &beep.Ctrl{
		Streamer: beep.Seq(p.volumeCtrl, beep.Callback(func() {
			done <- true
		})),
		Paused: false,
	}

	speaker.Play(p.ctrl)
	p.playing = true
	p.paused = false

	go func() {
		<-done
		p.mu.Lock()
		wasPlaying := p.playing
		p.mu.Unlock()

		if wasPlaying {
			p.HandleTrackEnd()
		}
	}()

	return nil
}

func (p *Player) Pause() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.ctrl != nil && p.playing {
		speaker.Lock()
		p.ctrl.Paused = true
		speaker.Unlock()
		p.paused = true
		p.playing = false
	}
}

func (p *Player) Resume() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.ctrl != nil && p.paused {
		speaker.Lock()
		p.ctrl.Paused = false
		speaker.Unlock()
		p.playing = true
		p.paused = false
	}
}

func (p *Player) Next() error {
	p.Stop()

	p.mu.Lock()
	playlist := p.getCurrentPlaylist()
	p.currentIndex = (p.currentIndex + 1) % len(playlist)

	if p.shuffle && p.currentIndex == 0 && p.repeatMode == RepeatAll {
		p.createShuffledPlaylist()
	}
	p.mu.Unlock()

	return p.Play()
}

func (p *Player) Previous() error {
	p.Stop()

	p.mu.Lock()
	playlist := p.getCurrentPlaylist()
	p.currentIndex = (p.currentIndex - 1 + len(playlist)) % len(playlist)
	p.mu.Unlock()

	return p.Play()
}

func (p *Player) Skip(index int) error {
	if index < 0 || index >= len(p.tracks) {
		return fmt.Errorf("index out of range")
	}

	p.Stop()

	p.mu.Lock()
	if p.shuffle && p.shuffledTracks != nil {
		targetTrack := p.tracks[index]
		for i, track := range p.shuffledTracks {
			if track.Path == targetTrack.Path {
				p.currentIndex = i
				break
			}
		}
	} else {
		p.currentIndex = index
	}
	p.mu.Unlock()

	return p.Play()
}

func (p *Player) ToggleShuffle() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.shuffle = !p.shuffle

	if p.shuffle {
		p.createShuffledPlaylist()

		if p.currentIndex < len(p.tracks) {
			currentTrack := p.tracks[p.currentIndex]
			for i, track := range p.shuffledTracks {
				if track.Path == currentTrack.Path {
					p.currentIndex = i
					break
				}
			}
		}
	} else {
		if p.shuffledTracks != nil && p.currentIndex < len(p.shuffledTracks) {
			currentTrack := p.shuffledTracks[p.currentIndex]
			for i, track := range p.tracks {
				if track.Path == currentTrack.Path {
					p.currentIndex = i
					break
				}
			}
		}
		p.shuffledTracks = nil
	}
}

func (p *Player) ToggleRepeat() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.repeatMode = (p.repeatMode + 1) % 3
}

func (p *Player) SetRepeatMode(mode RepeatMode) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.repeatMode = mode
}

func (p *Player) GetRepeatMode() RepeatMode {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.repeatMode
}

func (p *Player) HandleTrackEnd() error {
	p.mu.Lock()
	mode := p.repeatMode
	playlist := p.getCurrentPlaylist()
	p.mu.Unlock()

	switch mode {
	case RepeatOne:
		p.Stop()
		return p.Play()
	case RepeatAll:
		return p.Next()
	default:
		p.mu.Lock()
		if p.currentIndex < len(playlist)-1 {
			p.mu.Unlock()
			return p.Next()
		}
		p.mu.Unlock()
		p.Stop()
		return nil
	}
}

func (p *Player) GetCurrentTime() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.streamer != nil && p.playing {
		speaker.Lock()
		pos := p.streamer.Position()
		speaker.Unlock()
		p.currentTime = time.Duration(float64(pos) / float64(p.format.SampleRate) * float64(time.Second))
	}

	return p.currentTime
}

func (p *Player) GetTotalTime() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.totalTime
}

func (p *Player) GetCurrentTrack() Track {
	p.mu.Lock()
	defer p.mu.Unlock()

	playlist := p.getCurrentPlaylist()
	if p.currentIndex < len(playlist) {
		return playlist[p.currentIndex]
	}
	return Track{}
}

func (p *Player) IsPlaying() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.playing
}

func (p *Player) GetShuffle() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.shuffle
}

func (p *Player) GetCurrentIndex() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.currentIndex
}

func (p *Player) volumeToDecibels(volume float64) float64 {
	if volume <= 0.0001 {
		return -10
	}

	return (volume - 1.0) * 5.0
}

func (p *Player) SetVolume(volume float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if volume < 0 {
		volume = 0
	}

	if volume > 1 {
		volume = 1
	}

	p.volume = volume
	if p.volumeCtrl != nil {
		speaker.Lock()
		p.volumeCtrl.Volume = p.volumeToDecibels(volume)
		p.volumeCtrl.Silent = (volume < 0.0001)
		speaker.Unlock()
	}
}

func (p *Player) GetVolume() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.volume
}

func (p *Player) SeekForward() error {
	return p.seek(10 * time.Second)
}

func (p *Player) SeekBackward() error {
	return p.seek(-10 * time.Second)
}

func (p *Player) seek(offset time.Duration) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.streamer == nil || !p.playing {
		return fmt.Errorf("no track is currently playing")
	}

	speaker.Lock()
	defer speaker.Unlock()

	currentPos := p.streamer.Position()
	currentTime := time.Duration(float64(currentPos) / float64(p.format.SampleRate) * float64(time.Second))
	newTime := currentTime + offset

	if newTime < 0 {
		newTime = 0
	}

	if newTime > p.totalTime {
		newTime = p.totalTime
	}

	newPos := int(float64(newTime) / float64(time.Second) * float64(p.format.SampleRate))

	if newPos < 0 {
		newPos = 0
	}

	if newPos >= p.streamer.Len() {
		newPos = p.streamer.Len() - 1
	}

	if err := p.streamer.Seek(newPos); err != nil {
		return fmt.Errorf("failed to seek: %v", err)
	}

	p.currentTime = newTime
	return nil
}

func (p *Player) GetProgress() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.totalTime == 0 {
		return 0.0
	}

	if p.streamer != nil && p.playing {
		speaker.Lock()
		pos := p.streamer.Position()
		speaker.Unlock()
		p.currentTime = time.Duration(float64(pos) / float64(p.format.SampleRate) * float64(time.Second))
	}

	return float64(p.currentTime) / float64(p.totalTime)
}

func (p *Player) SeekToPosition(position float64) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.streamer == nil || !p.playing {
		return fmt.Errorf("no track is currently playing")
	}

	if position < 0.0 {
		position = 0.0
	}

	if position > 1.0 {
		position = 1.0
	}

	speaker.Lock()
	defer speaker.Unlock()

	newPos := int(position * float64(p.streamer.Len()))
	if err := p.streamer.Seek(newPos); err != nil {
		return fmt.Errorf("failed to seek: %v", err)
	}

	p.currentTime = time.Duration(float64(newPos) / float64(p.format.SampleRate) * float64(time.Second))
	return nil
}

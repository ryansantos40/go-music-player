package utils

import (
	"math/rand"
	"time"
	"os"
	"strings"
	"path/filepath"
	"fmt"


	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/wav"
	"github.com/faiface/beep/flac"
	"github.com/faiface/beep/speaker"
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
	tracks []Track
	currentIndex int
	playing bool
	paused  bool
	shuffle bool
	repeatMode RepeatMode
	streamer beep.StreamSeeker
	ctrl    *beep.Ctrl
	format  beep.Format
	currentTime time.Duration
	totalTime   time.Duration
}

func decodeAudioFile(f *os.File) (beep.StreamSeeker, beep.Format, error) {
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
		tracks: tracks,
		currentIndex: 0,
		playing: false,
		shuffle: false,
		repeatMode: RepeatOff,
	}
}

func (p *Player) Stop() {
	if p.ctrl != nil {
		p.ctrl.Streamer = nil
	}

	speaker.Clear()
	p.playing = false
	p.currentTime = 0
}

func (p *Player) Play() error {
	if p.currentIndex >= len(p.tracks) {
		return nil
	}

	track := p.tracks[p.currentIndex]
	f, err := os.Open(track.Path)
	if err != nil {
		return err
	}
	defer f.Close()

	streamer, format, err := decodeAudioFile(f)
	if err != nil {
		return err
	}

	p.streamer = streamer
	p.format = format
	p.ctrl = &beep.Ctrl{Streamer: streamer}
	p.totalTime = time.Duration(float64(streamer.Len()) / float64(format.SampleRate) * float64(time.Second))

	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	speaker.Play(p.ctrl)

	p.playing = true
	p.paused = false
	return nil
}

func (p *Player) Pause() {
	if p.ctrl != nil && p.playing {
		p.paused = true
		p.playing = false
	}
}

func (p *Player) Resume() {
	if p.ctrl != nil && p.paused {
		p.playing = true
		p.paused = false
	}
}

func (p *Player) Next() error {
	p.Stop()
	if p.shuffle {
		p.currentIndex = rand.Intn(len(p.tracks))
	} else {
		p.currentIndex = (p.currentIndex + 1) % len(p.tracks)
	}

	return p.Play()
}

func (p *Player) Previous() error {
	p.Stop()
	if p.shuffle {
		p.currentIndex = rand.Intn(len(p.tracks))

	} else {
		p.currentIndex = (p.currentIndex - 1 + len(p.tracks)) % len(p.tracks)
	}

	return p.Play()
}

func (p *Player) Skip(index int) error{
	if index < 0 || index >= len(p.tracks) {
		return fmt.Errorf("index out of range")
	}
	p.Stop()
	p.currentIndex = index
	return p.Play()
}

func (p *Player) ToggleShuffle() {
	p.shuffle = !p.shuffle
}

func (p *Player) ToggleRepeat() {
	p.repeatMode = (p.repeatMode + 1) % 3
}

func (p *Player) SetRepeatMode(mode RepeatMode) {
	p.repeatMode = mode
}

func (p *Player) GetRepeatMode() RepeatMode {
	return p.repeatMode
}

func (p *Player) HandleTrackEnd() error {
	switch p.repeatMode {
		case RepeatOne:
			p.Stop()
			return p.Play()

		case RepeatAll:
			return p.Next()
		
		default:
			return p.Next()
	}
}

func (p *Player) GetCurrentTime() time.Duration {
	if p.streamer != nil {
		pos := p.streamer.Position()
		p.currentTime = time.Duration(float64(pos) / float64(p.format.SampleRate) * float64(time.Second))
	}

	return p.currentTime
}

func (p *Player) GetTotalTime() time.Duration {
	return p.totalTime
}

func (p *Player) GetCurrentTrack() Track {
	if p.currentIndex < len(p.tracks) {
		return p.tracks[p.currentIndex]
	}
	return Track{}
}

func (p *Player) IsPlaying() bool {
	return p.playing
}

func (p *Player) GetShuffle() bool {
	return p.shuffle
}

func (p *Player) GetCurrentIndex() int {
	return p.currentIndex
}
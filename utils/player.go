package utils

import (
	"math/rand"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/wav"
	"github.com/faiface/beep/flac"
)

type Player struct {
	tracks []Track
	currentIndex Int
	playing bool
	shuffle bool
	streamer beep.StreamSeeker
	ctrl    *beep.Ctrl
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

	streamer, format, err := decodeAudioFile(f)
	if err != nil {
		return err
	}

	p.streamer = streamer
	p.ctrl = &beep.Ctrl{Streamer: beep.Loop(-1, streamer)}
	p.totalTime = format.SampleRate.D(stramer.Len()).Round(time.Second)

	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	speaker.Play(p.ctrl)

	p.playing = true
	return nil
}

func (p *Player) Pause() {
	if p.ctrl != nil {
		p.ctrl.Paused = true
	}
	p.playing = false
}

func (p *Player) Resume() {
	if p.ctrl != nil {
		p.ctrl.Paused = false
	}
	p.playing = true
}

func (p *Player) Next() {
	p.Stop()
	if p.shuffle {
		p.currentIndex = rand.Intn(len(p.tracks))
	} else {
		p.currentIndex = (p.currentIndex + 1) % len(p.tracks)
	}

	p.Play()
}

func (p *Player) Previous() {
	p.Stop()
	if p.shuffle {
		p.currentIndex = rand.Intn(len(p.tracks))

	} else {
		p.currentIndex = (p.currentIndex - 1 + len(p.tracks)) % len(p.tracks)
	}

	p.Play()
}

func (p *Player) Skip(index int) {
	if index >= 0 && index < len(p.tracks) {
		p.Stop()
		p.currentIndex = index
		p.Play()
	}
}

func (p *Player) ToggleShuffle() {
	p.shuffle = !p.shuffle
}

func (p *Player) GetCurrentTime() time.Duration {
	if p.stramer != nil {
		pos := p.streamer.Position()
		p.currentTime = p.stramer.Len().D(pos).Round(time.Second)
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
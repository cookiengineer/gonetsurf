package animations

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestParseKeyframes(t *testing.T) {
	keyframes, err := ParseKeyframesRule("@keyframes slide { from { left: 0 } 50% { left: 50px } to { left: 100px } }")
	if err != nil {
		t.Fatalf("ParseKeyframesRule: %v", err)
	}
	if keyframes.Name != "slide" {
		t.Errorf("name = %q, want slide", keyframes.Name)
	}
	if len(keyframes.Frames) != 3 {
		t.Fatalf("got %d frames, want 3", len(keyframes.Frames))
	}
	if keyframes.Frames[0].Offset != 0 || keyframes.Frames[1].Offset != 0.5 || keyframes.Frames[2].Offset != 1 {
		t.Errorf("offsets = %v", []float64{keyframes.Frames[0].Offset, keyframes.Frames[1].Offset, keyframes.Frames[2].Offset})
	}
	if len(keyframes.Frames[1].Declarations) != 1 || keyframes.Frames[1].Declarations[0].Name != "left" {
		t.Errorf("frame 1 declarations = %#v", keyframes.Frames[1].Declarations)
	}
}

func TestParseKeyframesSelectors(t *testing.T) {
	keyframes, err := ParseKeyframesRule("@keyframes multi { 0%, 100% { opacity: 1 } }")
	if err != nil {
		t.Fatalf("ParseKeyframesRule: %v", err)
	}
	if len(keyframes.Frames) != 2 {
		t.Fatalf("got %d frames, want 2", len(keyframes.Frames))
	}
	if keyframes.Frames[0].Offset != 0 || keyframes.Frames[1].Offset != 1 {
		t.Errorf("offsets = %v, %v", keyframes.Frames[0].Offset, keyframes.Frames[1].Offset)
	}
}

func TestParseAnimation(t *testing.T) {
	animation, err := ParseAnimation("slide 2s ease-in 1s infinite alternate")
	if err != nil {
		t.Fatalf("ParseAnimation: %v", err)
	}
	if animation.Name != "slide" {
		t.Errorf("name = %q", animation.Name)
	}
	if animation.Duration != 2*time.Second {
		t.Errorf("duration = %v", animation.Duration)
	}
	if animation.Delay != 1*time.Second {
		t.Errorf("delay = %v", animation.Delay)
	}
	if animation.TimingFunction != "ease-in" {
		t.Errorf("timing = %q", animation.TimingFunction)
	}
	if animation.IterationCount != -1 {
		t.Errorf("iteration = %v", animation.IterationCount)
	}
	if animation.Direction != "alternate" {
		t.Errorf("direction = %q", animation.Direction)
	}
}

type animationCase struct {
	Input string `json:"input"`
	Valid bool   `json:"valid"`
}

func TestAnimationConformance(t *testing.T) {
	encoded, err := os.ReadFile("testdata/animation.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []animationCase
	if err := json.Unmarshal(encoded, &cases); err != nil {
		t.Fatal(err)
	}
	for _, testCase := range cases {
		_, err := ParseAnimation(testCase.Input)
		if (err == nil) != testCase.Valid {
			t.Errorf("ParseAnimation(%q) valid = %v, want %v (err: %v)", testCase.Input, err == nil, testCase.Valid, err)
		}
	}
}

func TestAnimationList(t *testing.T) {
	animations, err := ParseAnimationList("anim paused both reverse, 4 1s -3s cubic-bezier(0, -2, 1, 3)")
	if err != nil {
		t.Fatalf("ParseAnimationList: %v", err)
	}
	if len(animations) != 2 {
		t.Fatalf("got %d animations, want 2", len(animations))
	}
	if animations[0].Name != "anim" || animations[1].IterationCount != 4 {
		t.Errorf("animations = %#v", animations)
	}
}

package main

import (
	"fmt"
	"image"
	"strings"
	"time"

	"github.com/akatsuki105/dawngb/agent"
)

type AI struct {
	vision  *agent.Vision
	planner *agent.Planner
	actor   *agent.Actor

	running  bool
	interval time.Duration

	lastInputUpdate time.Time
	inputStream     chan string
	input           map[string]bool
}

func NewAI() *AI {
	ai := &AI{}
	ai.running = false

	// models
	ai.vision = agent.NewVisionAgent()
	ai.planner = agent.NewPlannerAgent()
	ai.actor = agent.NewActorAgent()

	// decision speed
	ai.interval = time.Second * 10

	// input stuff
	ai.lastInputUpdate = time.Now()
	ai.inputStream = make(chan string)
	ai.input = map[string]bool{
		"A":      false,
		"B":      false,
		"START":  false,
		"SELECT": false,
		"UP":     false,
		"DOWN":   false,
		"LEFT":   false,
		"RIGHT":  false,
	}

	return ai
}

var last_input string

func (ai *AI) SetLatestFrame(img *image.RGBA) {
	// update AI brain with latest frame from GB
	ai.vision.SetImage(img) // nonblocking update last frame image in vision model
}

func (ai *AI) PollInput() map[string]bool {
	// if no input return false for all values
	if time.Since(ai.lastInputUpdate) > (time.Millisecond * 500) {
		// reset inputs
		ai.input = map[string]bool{
			"A":      false,
			"B":      false,
			"START":  false,
			"SELECT": false,
			"UP":     false,
			"DOWN":   false,
			"LEFT":   false,
			"RIGHT":  false,
		}
		// compute new input
		select {
		case input := <-ai.inputStream:
			fmt.Println("poll AI input wants to send " + input)
			ai.input[input] = true
		default:
		}
		ai.lastInputUpdate = time.Now()
	}

	return ai.input
}

func (ai *AI) Start() {
	if ai.running {
		return
	}
	ai.running = true

	// start running the AI brain ticketing
	fmt.Println("STARTING AI")
	ticker := time.NewTicker(ai.interval)

	for _ = range ticker.C {
		desc, err := ai.vision.DescribeScene()
		if err != nil {
			fmt.Printf("ERR: %s\n", err.Error())
			return
		}

		plan, err := ai.planner.Plan(desc, last_input)
		if err != nil {
			fmt.Printf("ERR: %s\n", err.Error())
			return
		}

		act, err := ai.actor.Act(desc, plan)
		if err != nil {
			fmt.Printf("ERR: %s\n", err.Error())
			return
		}

		last_input = strings.Join(act, ", ")

		fmt.Println(desc)
		fmt.Println("---")
		fmt.Println(plan)
		fmt.Println("---")
		fmt.Println(strings.Join(act, ", "))

		for _, input := range act {
			// TODO: catch if this is buffered lmao
			ai.inputStream <- input
		}
	}

}

package ui

import (
	"nookbuddy/internal/config"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

const (
	roomWidth  = 36
	roomHeight = 13

	workingFrameDuration = 500 * time.Millisecond
	snoreFrameDuration   = 800 * time.Millisecond
)

type layer uint8

const (
	layerWalls layer = iota
	layerItem
	layerCharacter
	layerSnore
)

var layerStyles = map[layer]lipgloss.Style{
	layerWalls:     roomWalls,
	layerItem:      roomItem,
	layerCharacter: roomCharacter,
	layerSnore:     roomSnore,
}

type cell struct {
	glyph rune
	layer layer
}

type canvas [roomHeight][roomWidth]cell

type sprite struct {
	row, col int
	lines    []string
}

var cosmeticSprites = map[string]sprite{
	"window":    {1, 3, []string{"╔══╦══╗", "║  ║  ║", "╚══╩══╝"}},
	"painting":  {1, 13, []string{"┌─────┐", "│ ~^~ │", "└─────┘"}},
	"bookshelf": {5, 1, []string{"┌───┐", "│▌▐▌│", "│▐▌▐│", "└───┘"}},
	"flower":    {5, 14, []string{" @ ", `\|/`, "[_]"}},
	"rug":       {11, 9, []string{"░▒▓▓▓▓▓▓▓▓▓▓▓▓▒░"}},
}

var furniture = []sprite{
	{5, 27, []string{"┌─────┐", "│▓▓▓▓▓│", "└──┬──┘"}},
	{8, 12, []string{strings.Repeat("▄", 22)}},
	{9, 13, []string{"█", "█"}},
	{9, 32, []string{"█", "█"}},
}

const characterRow, characterCol = 5, 20

var workingFrames = [3][]string{
	{"(o_o)", `/[ ]\`, "=^=^="},
	{"(o_o)", "/[ ]-", "=^= o"},
	{"(o.o)", `\[ ]/`, "^=^=^"},
}

var sleepingPose = []string{"     ", `\___/`, "(-_-)"}

var snoreFrames = [3]sprite{
	{4, 23, []string{"z"}},
	{3, 24, []string{"Z"}},
	{2, 23, []string{"Z Z"}},
}

func sleeping(now, lastInputAt time.Time) bool {
	return lastInputAt.IsZero() || now.Sub(lastInputAt) >= config.IdleThreshold
}

func workingFrame(now time.Time) int {
	return int(now.UnixMilli()/workingFrameDuration.Milliseconds()) % len(workingFrames)
}

func snoreFrame(now time.Time) int {
	return int(now.UnixMilli()/snoreFrameDuration.Milliseconds()) % len(snoreFrames)
}

func (c *canvas) draw(s sprite, l layer) {
	for i, line := range s.lines {
		for j, glyph := range []rune(line) {
			row, col := s.row+i, s.col+j
			if glyph == ' ' || row < 0 || row >= roomHeight || col < 0 || col >= roomWidth {
				continue
			}
			c[row][col] = cell{glyph: glyph, layer: l}
		}
	}
}

func newRoomCanvas() *canvas {
	var c canvas
	for row := range c {
		for col := range c[row] {
			c[row][col] = cell{glyph: ' ', layer: layerWalls}
		}
	}
	inner := strings.Repeat("─", roomWidth-2)
	c.draw(sprite{0, 0, []string{"┌" + inner + "┐"}}, layerWalls)
	c.draw(sprite{roomHeight - 1, 0, []string{"└" + inner + "┘"}}, layerWalls)
	for row := 1; row < roomHeight-1; row++ {
		c[row][0] = cell{glyph: '│', layer: layerWalls}
		c[row][roomWidth-1] = cell{glyph: '│', layer: layerWalls}
	}
	return &c
}

func renderScene(now, lastInputAt time.Time, owned []string) string {
	c := newRoomCanvas()
	for _, piece := range furniture {
		c.draw(piece, layerWalls)
	}
	for _, id := range owned {
		if s, ok := cosmeticSprites[id]; ok {
			c.draw(s, layerItem)
		}
	}

	if sleeping(now, lastInputAt) {
		c.draw(sprite{characterRow, characterCol, sleepingPose}, layerCharacter)
		c.draw(snoreFrames[snoreFrame(now)], layerSnore)
	} else {
		c.draw(sprite{characterRow, characterCol, workingFrames[workingFrame(now)]}, layerCharacter)
	}

	lines := make([]string, roomHeight)
	for row := range c {
		lines[row] = c.renderRow(row)
	}
	return strings.Join(lines, "\n")
}

func (c *canvas) renderRow(row int) string {
	var out, run strings.Builder
	current := c[row][0].layer
	for _, cl := range c[row] {
		if cl.layer != current {
			out.WriteString(layerStyles[current].Render(run.String()))
			run.Reset()
			current = cl.layer
		}
		run.WriteRune(cl.glyph)
	}
	out.WriteString(layerStyles[current].Render(run.String()))
	return out.String()
}

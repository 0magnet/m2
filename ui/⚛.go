// Package main ui/⚛.go
package main

import (
	"runtime"
	"strings"
	"unicode/utf8"
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/tree"
	"github.com/0magnet/calvin"
)

var home *core.Frame
var mySvg *core.SVG

var (
	title = strings.Replace(calvin.AsciiFont(siteName), " ", "\u00A0", -1)
	titleWidth int
	tagLine = "we have the technology"
)


func initTitle() {
	title = strings.Replace(calvin.AsciiFont(siteName), " ", "\u00A0", -1)

	titles := strings.Split(title, "\n")
	for i := 0; i < len(titles) && i < 3; i++ {
		titles[i] = " " + titles[i]
	}

	title = strings.Join(titles, "\n")
	titleWidth = utf8.RuneCountInString(strings.Split(title, "\n")[0])
}

func homePage(ctx *htmlcore.Context) bool {
	initTitle()
	home = core.NewFrame(ctx.BlockParent)
	home.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
		s.CenterAll()
	})
	home.OnShow(func(_ events.Event) {
		home.Update()
	})

	tree.AddChildAt(home, "", func(w *core.Frame) {
		w.Styler(func(s *styles.Style) {
			s.Gap.Set(units.Em(0))
			s.Grow.Set(1, 0)
			if home.SizeClass() == core.SizeCompact {
				s.Direction = styles.Column
			}
		})
		w.Maker(func(p *tree.Plan) {
			tree.Add(p, func(w *core.Frame) {
				w.Styler(func(s *styles.Style) {
					s.Direction = styles.Column
					s.Text.Align = text.Start
					s.Grow.Set(1, 1)
				})
			})
		})
	})

	tree.AddChild(home, func(w *core.Text) {
		w.SetType(core.TextHeadlineLarge).SetText(title)
		w.Styler(func(s *styles.Style) {
			s.Font.Family = rich.Custom
			if runtime.GOOS == "js" {
				s.Font.Family = rich.Monospace
			}
			s.Font.CustomFont = "mononoki"
			s.Text.WhiteSpace = text.WhiteSpacePre
			s.Min.X = units.Em(float32(titleWidth) * 0.62)
			s.Text.LineHeight = 1
			s.Gap.Set(units.Em(0))
			s.Color = colors.Scheme.Primary.Base
		})
	})

	//svg + canvas animation overlay
	tree.AddChild(home, func(w *core.Frame) {
		w.Styler(func(s *styles.Style) {
			s.Display = styles.Custom
			s.SetAbilities(true, abilities.Clickable)
			s.Grow.Set(1, 1)
		})

		w.OnClick(func(_ events.Event) {
			pause = !pause
			if pause {
				core.MessageSnackbar(b, "animation paused")
			} else {
				core.MessageSnackbar(b, "animation resumed")
			}
		})
		mySvg = core.NewSVG(w)
		errors.Log(mySvg.OpenFS(mySVG, "icon.svg")) //nolint
		mySvg.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 1)
		})
		c := core.NewCanvas(w)
		c.Styler(func(s *styles.Style) {
			s.RenderBox = false
			s.Grow.Set(1, 1)
		})
		globeAnimation(c)
	})

	tree.AddChild(home, func(w *core.Text) {
		w.SetType(core.TextTitleLarge).SetText(tagLine)

	})
	return true
}

// Copyright (c) 2026 John Dewey

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/retr0h/tonestack/pkg/cli"
)

// styleHelp makes every command print help in the project's own palette.
//
// Cobra renders help from a text template and offers no colour of its own.
// Replacing the renderer rather than the template keeps the layout in
// internal/tui, where it is tested, and leaves this as the adapter from
// cobra's types to it.
func styleHelp(root *cobra.Command) {
	root.SetHelpFunc(func(c *cobra.Command, _ []string) {
		out := c.OutOrStdout()

		// The banner names the tool, so it belongs on the page that is
		// reached by running the tool with nothing else to say — bare
		// invocation and --help alike, which both land here.
		if c == root {
			_, _ = fmt.Fprint(out, "\n"+cli.Banner(out))
		}

		_ = help(c).Render(out)
	})

	root.SetUsageFunc(func(c *cobra.Command) error {
		return help(c).Render(c.ErrOrStderr())
	})
}

// help describes a command in the terms the renderer takes.
func help(c *cobra.Command) cli.Help {
	h := cli.Help{
		Name:        name(c),
		Description: description(c),
		Usage:       c.UseLine(),
		Commands:    commands(c),
		Flags:       flags(c),
	}

	if c.HasAvailableSubCommands() {
		h.Usage = c.CommandPath() + " <command> [flags]"
		h.Footer = `Run "` + c.CommandPath() + ` <command> --help" for more about a command.`
	}

	return h
}

// name titles the page, except at the root where the banner has already
// said it.
func name(c *cobra.Command) string {
	if !c.HasParent() {
		return ""
	}

	return c.CommandPath()
}

// description prefers the long form, falling back to the short one.
func description(c *cobra.Command) string {
	if c.Long != "" {
		return c.Long
	}

	return c.Short
}

// commands lists the subcommands worth showing.
func commands(c *cobra.Command) []cli.Item {
	var items []cli.Item

	for _, sub := range c.Commands() {
		if !sub.IsAvailableCommand() {
			continue
		}

		items = append(items, cli.Item{Name: sub.Name(), Description: sub.Short})
	}

	return items
}

// flags lists a command's own flags, with the shorthand where it has one.
func flags(c *cobra.Command) []cli.Item {
	var items []cli.Item

	c.LocalFlags().VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}

		items = append(items, cli.Item{Name: flagName(f), Description: f.Usage})
	})

	return items
}

// flagName renders a flag the way it is typed, with its value's type.
//
// Boolean flags take no value, so naming a type for them would describe an
// invocation that does not work.
func flagName(f *pflag.Flag) string {
	var b strings.Builder

	if f.Shorthand != "" {
		b.WriteString("-" + f.Shorthand + ", ")
	} else {
		b.WriteString("    ")
	}

	b.WriteString("--" + f.Name)

	if f.Value.Type() != "bool" {
		b.WriteString(" " + f.Value.Type())
	}

	return b.String()
}

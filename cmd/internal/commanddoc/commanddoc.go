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
package commanddoc

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Render writes the reference for every command under root, root included.
//
// Commands come in the order cobra lists them, which is sorted, so the page
// does not change between runs for reasons nobody chose.
func Render(root *cobra.Command) []byte {
	var b bytes.Buffer

	b.WriteString(preamble)
	section(&b, root)

	return b.Bytes()
}

// section writes one command, then each command beneath it.
func section(b *bytes.Buffer, c *cobra.Command) {
	fmt.Fprintf(b, "\n## %s\n\n%s\n", c.CommandPath(), strings.TrimSpace(description(c)))

	usage := c.UseLine()
	if c.HasAvailableSubCommands() {
		usage = c.CommandPath() + " <command> [flags]"
	}

	fmt.Fprintf(b, "\n```text\n%s\n```\n", usage)

	subs := available(c)
	if len(subs) > 0 {
		b.WriteString("\n| command | what it does |\n| --- | --- |\n")

		for _, sub := range subs {
			fmt.Fprintf(b, "| [%s](#%s) | %s |\n", sub.Name(), anchor(sub), cell(sub.Short))
		}
	}

	if rows := flags(c); len(rows) > 0 {
		b.WriteString("\n| flag | takes | default | what it does |\n| --- | --- | --- | --- |\n")

		for _, row := range rows {
			b.WriteString(row)
		}
	}

	for _, sub := range subs {
		section(b, sub)
	}
}

// description prefers the long form, the way --help does.
func description(c *cobra.Command) string {
	if c.Long != "" {
		return c.Long
	}

	return c.Short
}

// available lists the subcommands somebody can run.
//
// Hidden ones stay hidden here too: a page is not the place to announce what
// the help deliberately leaves out.
func available(c *cobra.Command) []*cobra.Command {
	var out []*cobra.Command

	for _, sub := range c.Commands() {
		if sub.IsAvailableCommand() {
			out = append(out, sub)
		}
	}

	return out
}

// flags renders a command's own flags as table rows.
func flags(c *cobra.Command) []string {
	var rows []string

	c.LocalFlags().VisitAll(func(f *pflag.Flag) {
		if f.Hidden || f.Name == "help" {
			return
		}

		rows = append(rows, fmt.Sprintf("| `%s` | %s | %s | %s |\n",
			flagName(f), takes(f), defaultOf(f), cell(f.Usage)))
	})

	return rows
}

// flagName is the flag as it is typed, with its shorthand where it has one.
func flagName(f *pflag.Flag) string {
	if f.Shorthand != "" {
		return "-" + f.Shorthand + ", --" + f.Name
	}

	return "--" + f.Name
}

// takes names the value a flag expects. A boolean flag takes none.
func takes(f *pflag.Flag) string {
	if t := f.Value.Type(); t != "bool" {
		return t
	}

	return ""
}

// defaultOf shows a default only where it says something. An empty string,
// false and zero are what a flag left unset already means.
func defaultOf(f *pflag.Flag) string {
	switch f.DefValue {
	case "", "false", "0", "[]":
		return ""
	default:
		return "`" + f.DefValue + "`"
	}
}

// anchor is the fragment GitHub gives a heading naming this command.
func anchor(c *cobra.Command) string {
	return strings.ReplaceAll(c.CommandPath(), " ", "-")
}

// cell keeps text on one table row.
func cell(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")

	return strings.Join(strings.Fields(s), " ")
}

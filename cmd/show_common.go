package cmd

import (
	"bufio"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/fatih/color"
	"github.com/jaytaylor/html2text"
	"github.com/muesli/termenv"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/config"
	"github.com/zaquestion/lab/internal/git"
)

// maxPadding returns the max value of two string numbers
func maxPadding(xstr string, ystr string) int {
	x, _ := strconv.Atoi(xstr)
	y, _ := strconv.Atoi(ystr)
	if x > y {
		return len(xstr)
	}
	return len(ystr)
}

// printDiffLine does a color print of a diff lines.  Red lines are removals
// and green lines are additions.
func printDiffLine(strColor string, maxChars int, sOld string, sNew string, ltext string) {

	switch strColor {
	case "":
		fmt.Printf("%*s %*s %s\n", maxChars, sOld, maxChars, sNew, ltext)
	case "green":
		color.Green("%*s %*s %s\n", maxChars, sOld, maxChars, sNew, ltext)
	case "red":
		color.Red("%*s %*s %s\n", maxChars, sOld, maxChars, sNew, ltext)
	}
}

// displayDiff displays the diff referenced in a discussion
func displayDiff(diff string, chunkNum int, newLine int, oldLine int) {
	var (
		diffChunkNum int = -1
		oldLineNum   int = 0
		newLineNum   int = 0
		maxChars     int
	)

	scanner := bufio.NewScanner(strings.NewReader(diff))
	for scanner.Scan() {
		if regexp.MustCompile(`^@@`).MatchString(scanner.Text()) {
			diffChunkNum++
			s := strings.Split(scanner.Text(), " ")
			dOld := strings.Split(s[1], ",")
			dNew := strings.Split(s[2], ",")

			// The patch can have, for example, either
			// @@ -1 +1 @@
			// or
			// @@ -1272,6 +1272,8 @@
			// so (len - 1) makes sense in both cases.
			maxChars = maxPadding(dOld[len(dOld)-1], dNew[len(dNew)-1]) + 1
			if diffChunkNum == chunkNum {
				continue
			}
			if diffChunkNum > chunkNum {
				break
			}
		}

		if diffChunkNum == chunkNum {
			var (
				sOld string
				sNew string
			)
			strColor := ""
			ltext := scanner.Text()
			ltag := string(ltext[0])
			switch ltag {
			case " ":
				strColor = ""
				oldLineNum++
				newLineNum++
				sOld = strconv.Itoa(oldLineNum)
				sNew = strconv.Itoa(newLineNum)
			case "-":
				strColor = "red"
				oldLineNum++
				sOld = strconv.Itoa(oldLineNum)
				sNew = " "
			case "+":
				strColor = "green"
				newLineNum++
				sOld = " "
				sNew = strconv.Itoa(newLineNum)
			}

			// output line
			if mrShowNoColorDiff {
				strColor = ""
			}
			if newLine != 0 {
				if newLineNum <= newLine && newLineNum >= (newLine-4) {
					printDiffLine(strColor, maxChars, sOld, sNew, ltext)
				}
			} else if oldLineNum <= oldLine && oldLineNum >= (oldLine-4) {
				printDiffLine(strColor, maxChars, sOld, sNew, ltext)
			}
		}
	}
}

func displayCommitDiscussion(idNum int, note *gitlab.Note) {

	// The GitLab API only supports showing comments on the entire
	// changeset and not per-commit.  IOW, all diffs are shown against
	// HEAD.  This is confusing in some scenarios, however it's what the
	// API provides.

	// Get a unified diff for the entire file
	diff, err := git.GetUnifiedDiff(note.Position.BaseSHA, note.Position.HeadSHA, note.Position.OldPath, note.Position.NewPath)
	if err != nil {
		fmt.Printf("    Could not get unified diff: Execute 'lab mr checkout %d; git checkout master' and try again.\n", idNum)
		return
	}

	if diff == "" {
		fmt.Println("    Could not find 'git diff' command.")
		return
	}

	// In general, only have to display the NewPath, however there
	// are some unusual cases where the OldPath may be displayed
	if note.Position.NewPath == note.Position.OldPath {
		fmt.Println("File:" + note.Position.NewPath)
	} else {
		fmt.Println("Files[old:" + note.Position.OldPath + " new:" + note.Position.NewPath + "]")
	}

	displayDiff(diff, 0, note.Position.NewLine, note.Position.OldLine)
	fmt.Println("")
	fmt.Println("")
}

func getBoldStyle() ansi.StyleConfig {
	var style ansi.StyleConfig
	if termenv.HasDarkBackground() {
		style = glamour.DarkStyleConfig
	} else {
		style = glamour.LightStyleConfig
	}
	bold := true
	style.Document.Bold = &bold
	return style
}

func printDiscussions(discussions []*gitlab.Discussion, since string, idstr string, idNum int, renderMarkdown bool) {
	newAccessTime := time.Now().UTC()

	issueEntry := fmt.Sprintf("%s%d", idstr, idNum)
	// if specified on command line use that, o/w use config, o/w Now
	var (
		comparetime time.Time
		err         error
		sinceIsSet  = true
	)
	comparetime, err = dateparse.ParseLocal(since)
	if err != nil || comparetime.IsZero() {
		comparetime = getMainConfig().GetTime(CommandPrefix + issueEntry)
		if comparetime.IsZero() {
			comparetime = time.Now().UTC()
		}
		sinceIsSet = false
	}

	mdRendererNormal, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle())
	mdRendererBold, _ := glamour.NewTermRenderer(
		glamour.WithStyles(getBoldStyle()))

	// for available fields, see
	// https://godoc.org/github.com/xanzy/go-gitlab#Note
	// https://godoc.org/github.com/xanzy/go-gitlab#Discussion
	for _, discussion := range discussions {
		for i, note := range discussion.Notes {

			indentHeader, indentNote := "", ""
			commented := "commented"
			if !time.Time(*note.CreatedAt).Equal(time.Time(*note.UpdatedAt)) {
				commented = "updated comment"
			}

			if !discussion.IndividualNote {
				indentNote = "    "

				if i == 0 {
					commented = "started a discussion"
					if note.Position != nil {
						commented = "started a commit discussion"
					}
				} else {
					indentHeader = "    "
				}
			}

			noteBody := strings.Replace(note.Body, "\n", "<br>\n", -1)
			html2textOptions := html2text.Options{
				PrettyTables: true,
				OmitLinks:    true,
			}
			noteBody, _ = html2text.FromString(noteBody, html2textOptions)
			mdRenderer := mdRendererNormal
			printit := color.New().PrintfFunc()
			if note.System {
				splitNote := strings.SplitN(noteBody, "\n", 2)

				// system notes are informational messages only
				// and cannot have replies.  Do not output the
				// note.ID
				printit(`
* %s %s at %s
`,
					note.Author.Username, splitNote[0], time.Time(*note.UpdatedAt).String())
				if len(splitNote) == 2 {
					if renderMarkdown {
						splitNote[1], _ = mdRenderer.Render(splitNote[1])
					}
					printit(`%s
`,
						splitNote[1])
				}
				continue
			}

			printit(`
%s-----------------------------------
`,
				indentHeader)

			if time.Time(*note.UpdatedAt).After(comparetime) {
				mdRenderer = mdRendererBold
				printit = color.New(color.Bold).PrintfFunc()
			}

			if renderMarkdown {
				noteBody, _ = mdRenderer.Render(noteBody)
			}
			noteBody = strings.Replace(noteBody, "\n", "\n"+indentNote, -1)

			printit(`%s#%d: %s %s at %s

`,
				indentHeader, note.ID, note.Author.Username, commented, time.Time(*note.UpdatedAt).String())
			if note.Position != nil && i == 0 {
				displayCommitDiscussion(idNum, note)
			}
			printit(`%s%s
`,

				indentNote, noteBody)
		}
	}

	if sinceIsSet == false {
		config.WriteConfigEntry(CommandPrefix+issueEntry, newAccessTime, "", "")
	}
}

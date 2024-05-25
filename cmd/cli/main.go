package main

import (
	"agent"
	"downloader"
	"fmt"
	"os"
	"strings"
)

func main() {
	arguments, flags := parseArgs()
	if len(arguments) != 2 {
		usageAndExit(1)
	}
	command, url := arguments[0], arguments[1]

	agents := []downloader.Downloader{
		agent.NewBilibili(url, flags["sessdata"]),
	}
	var agent downloader.Downloader
	for _, a := range agents {
		if a.CanHandle(url) {
			agent = a
			break
		}
	}
	if agent == nil {
		fmt.Fprintf(os.Stderr, "No available agent to handle this url.")
		os.Exit(100)
	}

	switch command {
	case "info":
		info, err := agent.GetResourceInfo()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error occured when getting resource information. Error is: %v\n", err)
			os.Exit(101)
		}
		printInfo(info)
	case "download":
		progress := agent.Download(0, "")
		for p := range progress {
			if p.Err != nil {
				fmt.Fprintf(os.Stderr, "Failed to download. Error is: %v", p.Err)
				os.Exit(102)
			} else {
				printProgress(p)
			}
		}
		fmt.Println("")
	default:
		fmt.Fprintf(os.Stderr, "invalid command \"%s\"\n", command)
	}
}

func printProgress(p *downloader.Progress) {
	var sb strings.Builder
	sb.WriteString("  ")
	if len(p.Status) > 44 {
		sb.WriteString(p.Status[:40])
		sb.WriteString(" ...")
	} else {
		sb.WriteString(p.Status)
		sb.WriteString(strings.Repeat(" ", 44-len(p.Status)))
	}

	// progress bar
	left := int(p.Percentage * 20)
	if left == 20 {
		left = 19
	}
	sb.WriteRune('[')
	sb.WriteString(strings.Repeat("-", left))
	sb.WriteRune('>')
	sb.WriteString(strings.Repeat("-", 19-left))
	sb.WriteRune(']')
	var percentageStr string
	if p.Percentage == 1.0 {
		percentageStr = "100"
	} else {
		percentageStr = fmt.Sprintf("%.1f", p.Percentage*100)
	}
	if len(percentageStr) < 4 {
		percentageStr = strings.Repeat(" ", 4-len(percentageStr)) + percentageStr
	}
	sb.WriteString(fmt.Sprintf(" %s%%", percentageStr))

	fmt.Print(sb.String() + "\r")
}

func parseArgs() ([]string, map[string]string) {
	var flagName string
	var seenFlag bool
	arguments := make([]string, 0)
	flags := make(map[string]string)
	for i, arg := range os.Args[1:] {
		if arg == "--" {
			if seenFlag {
				flags[flagName] = "true"
			}
			arguments = append(arguments, os.Args[i+2:]...)
			break
		}
		if strings.HasPrefix(arg, "--") {
			if seenFlag {
				flags[flagName] = "true"
			}
			seenFlag = true
			flagName = arg[2:]
		} else if strings.HasPrefix(arg, "-") {
			if seenFlag {
				flags[flagName] = "true"
			}
			seenFlag = true
			if len(arg) > 2 {
				// multiple flags
				for _, r := range arg[1 : len(arg)-1] {
					flags[string(r)] = "true"
				}
				seenFlag = true
				flagName = arg[len(arg)-1:]
			}
		} else if seenFlag {
			flags[flagName] = arg
		} else {
			arguments = append(arguments, arg)
		}
	}
	return arguments, flags
}

func usageAndExit(exitCode int) {
	fmt.Fprintf(os.Stderr, "usage: %s <command> <url> [flags]", os.Args[0])
	os.Exit(exitCode)
}

func printInfo(info []downloader.ResourceInfo) {
	if len(info) == 1 {
		switch info[0].Type {
		case downloader.RT_Video:
			printVideoInfo(&info[0])
		default:

		}
	}
}

func printVideoInfo(info *downloader.ResourceInfo) {
	fmt.Printf("Site:                       %s\n", info.Site)
	fmt.Printf("Title:                      %s\n", info.Name)
	streamCnt := len(info.Streams) + len(info.DashStreams)
	if streamCnt == 0 {
		fmt.Println("Streams:                    !! No streams available !! Attaching authentication information may help.")
	} else {
		fmt.Println("Streams:                    Available quality and codecs:")
		if len(info.Streams) > 0 {
			fmt.Println("  [ Video ] ___________________________________")
			for _, s := range info.Streams {
				fmt.Printf("  - format:                 %s\n", s.Id)
				fmt.Printf("    container:              %s\n", s.Container)
				fmt.Printf("    size:                   %s\n", readableBytes(s.Size))
				fmt.Printf("    download with argument: %s\n", s.DownloadWith)
				for k, v := range s.Others {
					fmt.Printf("    %s:%s%s\n", k, strings.Repeat(" ", 23-len(k)), v)
				}
				fmt.Println("")
			}
		}
		if len(info.DashStreams) > 0 {
			fmt.Println("  [ Dash  ] ___________________________________")
			for _, s := range info.DashStreams {
				fmt.Printf("  - format:                 %s\n", s.Id)
				fmt.Printf("    container:              %s\n", s.Container)
				fmt.Printf("    size:                   %s\n", readableBytes(s.Size))
				fmt.Printf("    download with argument: %s\n", s.DownloadWith)
				for k, v := range s.Others {
					fmt.Printf("    %s:%s%s\n", k, strings.Repeat(" ", 23-len(k)), v)
				}
				fmt.Println("")
			}
		}
	}
}

const (
	kb float32 = 1 << (10 * (iota + 1))
	mb
	gb
	tb
)

func readableBytes(size int) string {
	sizeFloat := float32(size)
	switch {
	case sizeFloat > tb:
		return fmt.Sprintf("%.2fTB (%d bytes)", sizeFloat/tb, size)
	case sizeFloat > gb:
		return fmt.Sprintf("%.2fGB (%d bytes)", sizeFloat/gb, size)
	case sizeFloat > mb:
		return fmt.Sprintf("%.2fMB (%d bytes)", sizeFloat/mb, size)
	case sizeFloat > kb:
		return fmt.Sprintf("%.2fKB (%d bytes)", sizeFloat/kb, size)
	}

	return fmt.Sprintf("%d bytes", size)
}

package main

import (
	"os"
	"fmt"
	"bytes"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	STATE int = 0
	VOL_NAME string = ""
	DISK_DATA []byte
	PARTITION_INDEX int
	FILES_START uint32
	COLOR_NORM = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	COLOR_HIGH = lipgloss.NewStyle().Background(lipgloss.Color("#FFFFFF")).Foreground(lipgloss.Color("0"))
)

type Model struct {
	cursor int
	choices []string
	state int
}

func initialModel() Model {
	return Model {
		cursor:  0,
		choices: []string{"List partitions", "Select partition", "Exit"},
		state: 0,
	}
}

func saveFile() {
	os.WriteFile(os.Args[1], DISK_DATA, 0644)
}

var ChoiceList = [][]string {
	{}, // HOME: Variable since it depends on the number of volumes on that disk
	{"List files", "Delete file", "Rebuild BS", "Back"},
	{"Back"}, // list files
	{}, // Delete files
	{"Yes", "No"}, // Rebuild BS
}

var DiskChoices = []string {}

func (model Model) Init() tea.Cmd {
	return nil
}

func (model Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
		switch msg.(tea.KeyMsg).String() {
		case "left":
			if model.cursor > 0 {
				model.cursor--
			}
		case "right":
			if model.cursor < len(ChoiceList[STATE]) - 1 {
				model.cursor++
			}
		case "enter":
			switch STATE {
			case 0:
				switch model.cursor {
				case len(ChoiceList[0]) - 1:
					return model, tea.Quit
				default:
					found := false
					for i := 0; i < len(DISK_DATA); i++ {
						if bytes.HasPrefix(DISK_DATA[i:], []byte("LUFS")) {
							j := i + 8
							name := DISK_DATA[j:j + 16]	
							if bytes.HasPrefix(name[0:], []byte(VOL_NAME)) {
								PARTITION_INDEX = i
								found = true
								break
							}
						}
					}
					if found == false {
						fmt.Println("\033[31mError:\033[31m unknown error in finding volme")
						return model, tea.Quit
					}
					FILES_START = uint32(DISK_DATA[PARTITION_INDEX + 24]) << 24 | uint32(DISK_DATA[PARTITION_INDEX + 25]) << 16 | uint32(DISK_DATA[PARTITION_INDEX + 26]) << 8 | uint32(DISK_DATA[PARTITION_INDEX + 27])
					STATE = 1
					model.cursor = 0
				}
			case 1:
				switch model.cursor {
				case 0:
					STATE = 2
					model.cursor = 0
				case 2:
					STATE = 4
					model.cursor = 0
				case 3:
					STATE = 0
					model.cursor = 0
				}
			case 2:
				switch model.cursor {
				case 0:
					STATE = 1
					model.cursor = 0
				}
			case 4:
				switch model.cursor {
				case 0:
					ptr := 492
					for {
						if ptr >= len(DISK_DATA) {
							break
						}

						num := uint16(DISK_DATA[ptr]) << 8 | uint16(DISK_DATA[ptr + 1])
						if int(num) == 0 {
							DISK_DATA[ptr] = byte((PARTITION_INDEX + 32) >> 8)
							DISK_DATA[ptr + 1] = byte((PARTITION_INDEX + 32) & 0xFF)
							saveFile()
							break
						}
						
						ptr += 2

						if ptr >= 512 {
							break
						}
					}
					STATE = 1
					model.cursor = 0
				case 1:
					STATE = 1
					model.cursor = 0
				}	
			}
		}
	}
	return model, nil
}

func (model Model) View() string {
	s := "TestDisk (for LUFS)\nNovember 2025\nBy Alexander Flax\n\n"

	if STATE != 0 {
		size := uint32(DISK_DATA[PARTITION_INDEX + 4]) << 24 | uint32(DISK_DATA[PARTITION_INDEX + 5]) << 16 | uint32(DISK_DATA[PARTITION_INDEX + 6]) << 8 | uint32(DISK_DATA[PARTITION_INDEX + 7])
		s += "Partition label: " + string(DISK_DATA[PARTITION_INDEX + 7:PARTITION_INDEX + 7 + 16]) + "\n"
		s += "Partition size: " + fmt.Sprintf("0x%08x", size) + " bytes\n\n"	
	}

	if STATE == 2 {
		s += "\n"
		sysstart := FILES_START - 544
		s += "BOOT             | size: " + fmt.Sprintf("%d", PARTITION_INDEX) + " bytes\n"
		s += "SYSTEM           | size: " + fmt.Sprintf("%d", sysstart) + " bytes\n"
		ptr := FILES_START
		for {
			if ptr >= uint32(len(DISK_DATA)) {
				break
			}

			if bytes.HasPrefix(DISK_DATA[ptr:], []byte("LUFS")) && ptr + 24 < uint32(len(DISK_DATA)) {
				ptr += 4
				s += string(DISK_DATA[ptr:ptr + 16]) + " | "
				ptr += 16
				size := uint32(DISK_DATA[ptr]) << 24 | uint32(DISK_DATA[ptr + 1]) << 16 | uint32(DISK_DATA[ptr + 2]) << 8 | uint32(DISK_DATA[ptr + 3])
				s += "size: " + fmt.Sprintf("%d", size) + " bytes"
				ptr += 4
			} else {
				ptr++
			}
			s += "\n"
		}
		s += "\n"
	}

	if STATE == 4 {
		s += "Are you sure you want to rebuild the boot sector?\n"
	}

	for i := range ChoiceList[STATE] {
		if i == model.cursor {
			s += COLOR_HIGH.Render(" [ " + ChoiceList[STATE][i] + " ] ")
		} else {
			s += COLOR_NORM.Render(" [ " + ChoiceList[STATE][i] + " ] ")
		}
		s += " "
	}

	return s
}

func main() {	
	if len(os.Args) < 2 {
		fmt.Println("Usage: testdisk-lufs <disk image>")
		os.Exit(1)
	}

	data, err := os.ReadFile(os.Args[1])	
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	DISK_DATA = data
	// Write disk partitions to the choice list

	for i := 0; i < len(data); i++ {
		if bytes.HasPrefix(data[i:], []byte("LUFS")) {
			ChoiceList[0] = append(ChoiceList[0], "Partition '" + string(data[i + 8:i + 24]) + "' at offset " + fmt.Sprintf("0x%08x", i))
			DiskChoices = append(DiskChoices, string(data[i + 8:i + 24]))
		}
	}
	ChoiceList[0] = append(ChoiceList[0], "Exit")

	if err = tea.NewProgram(initialModel()).Start(); err != nil {
		panic(err)
	}
	os.Exit(0)	
}

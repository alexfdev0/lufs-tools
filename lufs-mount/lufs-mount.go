package main

import (
	"fmt"
	"os"	
	"bytes"
	"strings"
	fuse "github.com/winfsp/cgofuse/fuse"
)

var DiskName string
var Partition string
var data []byte
var index uint32
var fstart uint32

type File struct {
	Name string
	Size uint32
	Offset uint32
}

func ReadDisk() {
	data_, err := os.ReadFile(DiskName)
	data = data_
	if err != nil {	
		return
	}

	var ptr uint32
	for {
		if ptr >= uint32(len(data)) {
			break
		}

		if bytes.HasPrefix(data[ptr:], []byte("LUFS")) {
			if bytes.HasPrefix(data[ptr + 8:ptr + 8 + 16], []byte(Partition)) {
				fstart = uint32(data[ptr + 24]) << 24 | uint32(data[ptr + 25]) << 16 | uint32(data[ptr + 26]) << 8 | uint32(data[ptr + 27])
				index = ptr
				break
			} else {
				size := uint32(data[ptr + 4]) << 24 | uint32(data[ptr + 5]) << 16 | uint32(data[ptr + 6]) << 8 | uint32(data[ptr + 7])
				ptr += size
			}
		} else {
			ptr++
		}
	}	
}

func FileNameTranslation(name string) string {
	name_ := ""
	ext := ""

	next := 0
	for i := 0; i < len(name); i++ {
		if name[i] == ' ' {
			next = i + 1
			break
		}
		name_ = name_ + string(name[i])
	}
	for i := next; i < len(name); i++ {
		if name[i] == ' ' {
			break
		}
		ext = ext + string(name[i])
	}

	final := name_
	if ext != "" {
		final = final + "." + ext
	}
	return final
}

func CleanBuffer(data []byte) []byte {
	var realBuf []byte
	for i := 0; i < len(data); i++ {
		if data[i] == 0x00 {
			notend := false
			for j := i + 1; j < len(data); j++ {
				if data[j] != 0x00 {
					notend = true
					break
				}
			}
			if notend == false {
				break
			} else {
				realBuf = append(realBuf, data[i])
			}
		} else {
			realBuf = append(realBuf, data[i])
		}
	}
	return realBuf
}

func ReadAllFiles() []File {
	ptr := fstart

	Files := []File{}
	for {
		if ptr >= uint32(len(data)) {
			break
		}

		if bytes.HasPrefix(data[ptr:], []byte("LFSF")) {
			name := data[ptr + 4:ptr + 20]
			size := uint32(data[ptr + 20]) << 24 | uint32(data[ptr + 21]) << 16 | uint32(data[ptr + 22]) << 8 | uint32(data[ptr + 23])
			Files = append(Files, File{Name: FileNameTranslation(string(name)), Size: size, Offset: ptr})
			ptr += size
		} else {
			ptr++
		}
	}
	return Files
}

func ReturnFile(name string) File {
	switch name {
	case "BOOT":
		return File{Name: "BOOT", Size: 1024, Offset: 0}
	case "SYSTEM":
		return File{Name: "SYSTEM", Size: fstart - (index + 32), Offset: 1024}
	default:
		ptr := fstart
		for {
			if ptr >= uint32(len(data)) {
				break
			}

			if bytes.HasPrefix(data[ptr:], []byte("LFSF")) {
				name_ := data[ptr + 4:ptr + 20]
				size := uint32(data[ptr + 20]) << 24 | uint32(data[ptr + 21]) << 16 | uint32(data[ptr + 22]) << 8 | uint32(data[ptr + 23])
				if FileNameTranslation(string(name_)) == name {	
					return File{Name: FileNameTranslation(string(name_)), Size: size, Offset: ptr}	
				} else {
					ptr += size
				}
			} else {
				ptr++
			}
		}
		
		return File{Name: "__NOTFOUND"}
	}
}

type LUFS struct {
	fuse.FileSystemBase
}

func (fs *LUFS) Getattr(path string, stat *fuse.Stat_t, fh uint64) int {
	if path == "/" {
		stat.Mode = fuse.S_IFDIR | 0755
		return 0
	}
	pathReal := strings.TrimPrefix(path, "/")

	File := ReturnFile(pathReal)
	if File.Name != "__NOTFOUND" {
		stat.Mode = fuse.S_IFREG | 0644
		stat.Size = int64(File.Size)
		return 0
	}
	return -fuse.ENOENT	
}

func (fs *LUFS) Readdir(path string, fill func(name string, stat *fuse.Stat_t, off int64) bool, offset int64, fh uint64) int {
	if path == "/" {
		fill(".", nil, 0)
		fill("..", nil, 0)
		fill("BOOT", nil, 0)
		fill("SYSTEM", nil, 0)
		Files := ReadAllFiles()
		for i := 0; i < len(Files); i++ {
			File := Files[i]
			fill(File.Name, nil, 0)
		}
		return 0;
	}
	return -fuse.ENOENT
} 

func (fs *LUFS) Open(path string, flags int) (int, uint64) {
	return 0, 0
}

func (fs *LUFS) Read(path string, buff []byte, ofst int64, fh uint64) int {	
	pathReal := strings.TrimPrefix(path, "/")

	File := ReturnFile(pathReal)

	if File.Name != "__NOTFOUND" {
		ptr := int64(File.Offset + 24) + ofst
		fileData := data[ptr:ptr + int64(File.Size)]
		fileData = CleanBuffer(fileData)

		copied := 0
		for i := 0; i < len(buff); i++ {
			if ofst + int64(i) >= int64(len(fileData)) {
				break
			}
			buff[i] = fileData[int64(i) + ofst]
			copied++
		}
		return copied	
	}
	return 0
} 

var MOUNT string

func FUSEInit() {
	fs := &LUFS{}
	host := fuse.NewFileSystemHost(fs)
	host.Mount(MOUNT, nil)
}

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: lufs-mount <disk> <mountpoint> <partition>")
		os.Exit(1)
	}

	disk_image := os.Args[1]
	DiskName = disk_image
	mountpoint := os.Args[2]
	MOUNT = mountpoint
	partition := os.Args[3]
	Partition = partition

	data, err := os.ReadFile(disk_image)
	if err != nil {
		fmt.Println("Error reading disk: ", err)
		os.Exit(1)
	}

	var ptr uint32
	found := false
	for {
		if ptr >= uint32(len(data)) {
			break
		}

		if bytes.HasPrefix(data[ptr:], []byte("LUFS")) {
			if bytes.HasPrefix(data[ptr + 8:ptr + 8 + 16], []byte(partition)) {	
				found = true
				break
			} else {
				size := uint32(data[ptr + 4]) << 24 | uint32(data[ptr + 5]) << 16 | uint32(data[ptr + 6]) << 8 | uint32(data[ptr + 7])
				ptr += size
			}
		} else {
			ptr++
		}
	}
	if found == false {
		fmt.Println("Could not find partition '" + partition + "'")
		os.Exit(1)
	}
	ReadDisk()

	FUSEInit()	
}

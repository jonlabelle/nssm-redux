package versioninfo

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateResourceObject(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		goarch    string
		machine   uint16
		relocType uint16
	}{
		{
			name:      "amd64",
			goarch:    "amd64",
			machine:   imageFileMachineAMD64,
			relocType: imageRelAMD64Addr32NB,
		},
		{
			name:      "arm64",
			goarch:    "arm64",
			machine:   imageFileMachineARM64,
			relocType: imageRelARM64Addr32NB,
		},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cfg := Config{
				Comments:         "Managed service wrapper",
				CompanyName:      "Example Co",
				FileDescription:  "nssmr test binary",
				LegalCopyright:   "Copyright (c) Example Co",
				OriginalFilename: "nssmr.exe",
				ProductName:      "nssmr",
			}
			cfg.ApplyDefaults("v1.2.3.4", "nssmr.exe")

			machine, err := MachineForGOARCH(testCase.goarch)
			if err != nil {
				t.Fatalf("MachineForGOARCH(%q): %v", testCase.goarch, err)
			}

			data, err := Generate(cfg, machine)
			if err != nil {
				t.Fatalf("Generate(%q): %v", testCase.goarch, err)
			}

			filename := filepath.Join(t.TempDir(), "versioninfo.syso")
			if err := os.WriteFile(filename, data, 0o644); err != nil {
				t.Fatalf("write temp resource object: %v", err)
			}

			file, err := pe.Open(filename)
			if err != nil {
				t.Fatalf("pe.Open(%q): %v", filename, err)
			}
			defer file.Close()

			if file.FileHeader.Machine != testCase.machine {
				t.Fatalf("machine = %#x, want %#x", file.FileHeader.Machine, testCase.machine)
			}

			section := file.Section(".rsrc")
			if section == nil {
				t.Fatal("missing .rsrc section")
			}
			if len(section.Relocs) != 1 {
				t.Fatalf(".rsrc relocations = %d, want 1", len(section.Relocs))
			}
			if section.Relocs[0].Type != testCase.relocType {
				t.Fatalf("relocation type = %#x, want %#x", section.Relocs[0].Type, testCase.relocType)
			}
			if section.Relocs[0].VirtualAddress != resourceDataEntryOffset {
				t.Fatalf("relocation VA = %d, want %d", section.Relocs[0].VirtualAddress, resourceDataEntryOffset)
			}

			sectionData, err := section.Data()
			if err != nil {
				t.Fatalf("read .rsrc section: %v", err)
			}

			for _, expected := range []string{
				"Managed service wrapper",
				"Example Co",
				"nssmr test binary",
				"nssmr.exe",
				"v1.2.3.4",
			} {
				needle, _, err := utf16LE(expected, true)
				if err != nil {
					t.Fatalf("utf16LE(%q): %v", expected, err)
				}
				if !bytes.Contains(sectionData, needle) {
					t.Fatalf("resource section does not contain UTF-16 string %q", expected)
				}
			}

			signature := []byte{0xbd, 0x04, 0xef, 0xfe}
			offset := bytes.Index(sectionData, signature)
			if offset < 0 {
				t.Fatal("missing VS_FIXEDFILEINFO signature")
			}

			fileVersionMS := binary.LittleEndian.Uint32(sectionData[offset+8:])
			fileVersionLS := binary.LittleEndian.Uint32(sectionData[offset+12:])
			if fileVersionMS != fixedVersionWord(1, 2) {
				t.Fatalf("fileVersionMS = %#x, want %#x", fileVersionMS, fixedVersionWord(1, 2))
			}
			if fileVersionLS != fixedVersionWord(3, 4) {
				t.Fatalf("fileVersionLS = %#x, want %#x", fileVersionLS, fixedVersionWord(3, 4))
			}
		})
	}
}

func TestParseFixedVersion(t *testing.T) {
	t.Parallel()

	got, err := parseFixedVersion("v2.10.3-rc1")
	if err != nil {
		t.Fatalf("parseFixedVersion: %v", err)
	}

	want := fixedVersion{2, 10, 3, 0}
	if got != want {
		t.Fatalf("parseFixedVersion returned %#v, want %#v", got, want)
	}
}

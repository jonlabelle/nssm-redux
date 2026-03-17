package versioninfo

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf16"
)

const (
	defaultLangID    = 0x0409
	defaultCharsetID = 0x04b0

	imageFileMachineAMD64 = 0x8664
	imageFileMachineARM64 = 0xaa64

	imageRelAMD64Addr32NB = 0x0003
	imageRelARM64Addr32NB = 0x0002

	imageScnCntInitializedData = 0x00000040
	imageScnMemRead            = 0x40000000
	imageSymClassStatic        = 0x03

	versionResourceType = 16
	versionResourceID   = 1

	resourceDataEntryOffset = 72
	resourceDataOffset      = 88

	fileFlagsMask   = 0x0000003f
	fileFlagPrivate = 0x00000008
	fileFlagSpecial = 0x00000020
	fileOSWindows32 = 0x00040004
	fileTypeApp     = 0x00000001
)

// Config describes the VERSIONINFO fields to embed in a Windows binary.
type Config struct {
	LangID              uint16 `json:"langId"`
	CharsetID           uint16 `json:"charsetId"`
	FileVersion         string `json:"fileVersion"`
	ProductVersion      string `json:"productVersion"`
	FixedFileVersion    string `json:"fixedFileVersion"`
	FixedProductVersion string `json:"fixedProductVersion"`
	Comments            string `json:"comments"`
	CompanyName         string `json:"companyName"`
	FileDescription     string `json:"fileDescription"`
	InternalName        string `json:"internalName"`
	LegalCopyright      string `json:"legalCopyright"`
	LegalTrademarks     string `json:"legalTrademarks"`
	OriginalFilename    string `json:"originalFilename"`
	PrivateBuild        string `json:"privateBuild"`
	ProductName         string `json:"productName"`
	SpecialBuild        string `json:"specialBuild"`
}

// Machine identifies the COFF machine and relocation type used by a .syso file.
type Machine struct {
	GOARCH         string
	COFFMachine    uint16
	RelocationType uint16
}

var machineByGOARCH = map[string]Machine{
	"amd64": {
		GOARCH:         "amd64",
		COFFMachine:    imageFileMachineAMD64,
		RelocationType: imageRelAMD64Addr32NB,
	},
	"arm64": {
		GOARCH:         "arm64",
		COFFMachine:    imageFileMachineARM64,
		RelocationType: imageRelARM64Addr32NB,
	},
}

type fixedVersion [4]uint16

type versionString struct {
	Key   string
	Value string
}

type versionNode struct {
	Key         string
	Type        uint16
	Value       []byte
	ValueLength uint16
	Children    []versionNode
}

type coffFileHeader struct {
	Machine              uint16
	NumberOfSections     uint16
	TimeDateStamp        uint32
	PointerToSymbolTable uint32
	NumberOfSymbols      uint32
	SizeOfOptionalHeader uint16
	Characteristics      uint16
}

type coffSectionHeader struct {
	Name                 [8]byte
	VirtualSize          uint32
	VirtualAddress       uint32
	SizeOfRawData        uint32
	PointerToRawData     uint32
	PointerToRelocations uint32
	PointerToLineNumbers uint32
	NumberOfRelocations  uint16
	NumberOfLineNumbers  uint16
	Characteristics      uint32
}

type coffRelocation struct {
	VirtualAddress   uint32
	SymbolTableIndex uint32
	Type             uint16
}

type coffSymbol struct {
	Name               [8]byte
	Value              uint32
	SectionNumber      int16
	Type               uint16
	StorageClass       uint8
	NumberOfAuxSymbols uint8
}

type coffSectionAuxSymbol struct {
	Size           uint32
	NumRelocs      uint16
	NumLineNumbers uint16
	Checksum       uint32
	SecNum         uint16
	Selection      uint8
	_              [3]byte
}

type fixedFileInfo struct {
	Signature        uint32
	StrucVersion     uint32
	FileVersionMS    uint32
	FileVersionLS    uint32
	ProductVersionMS uint32
	ProductVersionLS uint32
	FileFlagsMask    uint32
	FileFlags        uint32
	FileOS           uint32
	FileType         uint32
	FileSubtype      uint32
	FileDateMS       uint32
	FileDateLS       uint32
}

// LoadConfig reads VERSIONINFO settings from a JSON file.
func LoadConfig(path string) (Config, error) {
	if path == "" {
		return Config{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read version info config %q: %w", path, err)
	}

	var cfg Config
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse version info config %q: %w", path, err)
	}

	return cfg, nil
}

// ApplyDefaults fills in omitted VERSIONINFO fields using the build version and output filename.
func (c *Config) ApplyDefaults(buildVersion, defaultFilename string) {
	if c.LangID == 0 {
		c.LangID = defaultLangID
	}
	if c.CharsetID == 0 {
		c.CharsetID = defaultCharsetID
	}
	if c.FileVersion == "" {
		c.FileVersion = buildVersion
	}
	if c.ProductVersion == "" {
		c.ProductVersion = buildVersion
	}

	baseName := strings.TrimSuffix(defaultFilename, filepath.Ext(defaultFilename))
	if c.InternalName == "" {
		c.InternalName = baseName
	}
	if c.ProductName == "" {
		c.ProductName = c.InternalName
	}
	if c.OriginalFilename == "" {
		c.OriginalFilename = defaultFilename
	}
}

// MachineForGOARCH returns the COFF settings for a supported Windows architecture.
func MachineForGOARCH(goarch string) (Machine, error) {
	machine, ok := machineByGOARCH[goarch]
	if !ok {
		return Machine{}, fmt.Errorf("unsupported Windows architecture %q", goarch)
	}
	return machine, nil
}

// Generate builds a COFF object file containing a VERSIONINFO resource.
func Generate(cfg Config, machine Machine) ([]byte, error) {
	fileVersion, err := parseFixedVersion(firstNonEmpty(cfg.FixedFileVersion, cfg.FileVersion))
	if err != nil {
		return nil, fmt.Errorf("parse fixed file version: %w", err)
	}
	productVersion, err := parseFixedVersion(firstNonEmpty(cfg.FixedProductVersion, cfg.ProductVersion))
	if err != nil {
		return nil, fmt.Errorf("parse fixed product version: %w", err)
	}

	versionInfo, err := buildVersionInfo(cfg, fileVersion, productVersion)
	if err != nil {
		return nil, err
	}

	resourceSection := buildResourceSection(cfg, versionInfo)
	return buildCOFFObject(machine, resourceSection)
}

func buildVersionInfo(cfg Config, fileVersion, productVersion fixedVersion) ([]byte, error) {
	fixedInfo, err := encodeFixedFileInfo(cfg, fileVersion, productVersion)
	if err != nil {
		return nil, err
	}

	stringNodes, err := makeStringNodes(cfg)
	if err != nil {
		return nil, err
	}

	stringTable := versionNode{
		Key:      fmt.Sprintf("%04X%04X", cfg.LangID, cfg.CharsetID),
		Type:     1,
		Children: stringNodes,
	}

	translationValue := make([]byte, 0, 4)
	translationValue = appendLE16(translationValue, cfg.LangID)
	translationValue = appendLE16(translationValue, cfg.CharsetID)

	root := versionNode{
		Key:         "VS_VERSION_INFO",
		Type:        0,
		Value:       fixedInfo,
		ValueLength: uint16(len(fixedInfo)),
		Children: []versionNode{
			{
				Key:      "StringFileInfo",
				Type:     1,
				Children: []versionNode{stringTable},
			},
			{
				Key:  "VarFileInfo",
				Type: 0,
				Children: []versionNode{
					{
						Key:         "Translation",
						Type:        0,
						Value:       translationValue,
						ValueLength: uint16(len(translationValue)),
					},
				},
			},
		},
	}

	return root.encode()
}

func makeStringNodes(cfg Config) ([]versionNode, error) {
	fields := []versionString{
		{Key: "Comments", Value: cfg.Comments},
		{Key: "CompanyName", Value: cfg.CompanyName},
		{Key: "FileDescription", Value: cfg.FileDescription},
		{Key: "FileVersion", Value: cfg.FileVersion},
		{Key: "InternalName", Value: cfg.InternalName},
		{Key: "LegalCopyright", Value: cfg.LegalCopyright},
		{Key: "LegalTrademarks", Value: cfg.LegalTrademarks},
		{Key: "OriginalFilename", Value: cfg.OriginalFilename},
		{Key: "PrivateBuild", Value: cfg.PrivateBuild},
		{Key: "ProductName", Value: cfg.ProductName},
		{Key: "ProductVersion", Value: cfg.ProductVersion},
		{Key: "SpecialBuild", Value: cfg.SpecialBuild},
	}

	nodes := make([]versionNode, 0, len(fields))
	for _, field := range fields {
		if field.Value == "" {
			continue
		}
		encoded, units, err := utf16LE(field.Value, true)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, versionNode{
			Key:         field.Key,
			Type:        1,
			Value:       encoded,
			ValueLength: uint16(units),
		})
	}

	return nodes, nil
}

func encodeFixedFileInfo(cfg Config, fileVersion, productVersion fixedVersion) ([]byte, error) {
	flags := uint32(0)
	if cfg.PrivateBuild != "" {
		flags |= fileFlagPrivate
	}
	if cfg.SpecialBuild != "" {
		flags |= fileFlagSpecial
	}

	info := fixedFileInfo{
		Signature:        0xfeef04bd,
		StrucVersion:     0x00010000,
		FileVersionMS:    fixedVersionWord(fileVersion[0], fileVersion[1]),
		FileVersionLS:    fixedVersionWord(fileVersion[2], fileVersion[3]),
		ProductVersionMS: fixedVersionWord(productVersion[0], productVersion[1]),
		ProductVersionLS: fixedVersionWord(productVersion[2], productVersion[3]),
		FileFlagsMask:    fileFlagsMask,
		FileFlags:        flags,
		FileOS:           fileOSWindows32,
		FileType:         fileTypeApp,
	}

	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, info); err != nil {
		return nil, fmt.Errorf("encode fixed file info: %w", err)
	}
	return buf.Bytes(), nil
}

func (n versionNode) encode() ([]byte, error) {
	data := make([]byte, 0, 256)
	data = append(data, 0, 0)
	data = appendLE16(data, n.ValueLength)
	data = appendLE16(data, n.Type)

	key, _, err := utf16LE(n.Key, true)
	if err != nil {
		return nil, err
	}
	data = append(data, key...)
	data = padTo4(data)
	data = append(data, n.Value...)

	if len(n.Children) > 0 {
		data = padTo4(data)
		for index, child := range n.Children {
			if index > 0 {
				data = padTo4(data)
			}
			childData, err := child.encode()
			if err != nil {
				return nil, err
			}
			data = append(data, childData...)
		}
	}

	if len(data) > 0xffff {
		return nil, fmt.Errorf("version resource block %q exceeds 65535 bytes", n.Key)
	}

	binary.LittleEndian.PutUint16(data[:2], uint16(len(data)))
	return data, nil
}

func buildResourceSection(cfg Config, versionInfo []byte) []byte {
	data := make([]byte, 0, resourceDataOffset+len(versionInfo))

	data = appendResourceDirectory(data, 1)
	data = appendResourceDirectoryEntry(data, versionResourceType, 24, true)

	data = appendResourceDirectory(data, 1)
	data = appendResourceDirectoryEntry(data, versionResourceID, 48, true)

	data = appendResourceDirectory(data, 1)
	data = appendResourceDirectoryEntry(data, cfg.LangID, resourceDataEntryOffset, false)

	data = appendLE32(data, resourceDataOffset)
	data = appendLE32(data, uint32(len(versionInfo)))
	data = appendLE32(data, 0)
	data = appendLE32(data, 0)

	data = append(data, versionInfo...)
	return data
}

func buildCOFFObject(machine Machine, resourceSection []byte) ([]byte, error) {
	dataOffset := binary.Size(coffFileHeader{}) + binary.Size(coffSectionHeader{})
	relocationOffset := dataOffset + len(resourceSection)
	symbolOffset := relocationOffset + binary.Size(coffRelocation{})

	header := coffFileHeader{
		Machine:              machine.COFFMachine,
		NumberOfSections:     1,
		PointerToSymbolTable: uint32(symbolOffset),
		NumberOfSymbols:      2,
	}

	section := coffSectionHeader{
		Name:                 coffName(".rsrc"),
		VirtualSize:          uint32(len(resourceSection)),
		SizeOfRawData:        uint32(len(resourceSection)),
		PointerToRawData:     uint32(dataOffset),
		PointerToRelocations: uint32(relocationOffset),
		NumberOfRelocations:  1,
		Characteristics:      imageScnCntInitializedData | imageScnMemRead,
	}

	relocation := coffRelocation{
		VirtualAddress:   resourceDataEntryOffset,
		SymbolTableIndex: 0,
		Type:             machine.RelocationType,
	}

	symbol := coffSymbol{
		Name:               coffName(".rsrc"),
		SectionNumber:      1,
		StorageClass:       imageSymClassStatic,
		NumberOfAuxSymbols: 1,
	}

	aux := coffSectionAuxSymbol{
		Size:      uint32(len(resourceSection)),
		NumRelocs: 1,
	}

	var buf bytes.Buffer
	for _, value := range []any{header, section, resourceSection, relocation, symbol, aux} {
		if err := binary.Write(&buf, binary.LittleEndian, value); err != nil {
			return nil, fmt.Errorf("encode COFF resource object: %w", err)
		}
	}
	if err := binary.Write(&buf, binary.LittleEndian, uint32(4)); err != nil {
		return nil, fmt.Errorf("encode COFF string table: %w", err)
	}

	return buf.Bytes(), nil
}

func parseFixedVersion(value string) (fixedVersion, error) {
	var version fixedVersion
	trimmed := strings.TrimSpace(value)
	trimmed = strings.TrimPrefix(trimmed, "v")
	trimmed = strings.TrimPrefix(trimmed, "V")
	if trimmed == "" {
		return version, nil
	}

	var prefix strings.Builder
	seenDigit := false
	seenDot := false
	stoppedEarly := false
	for _, r := range trimmed {
		switch {
		case r >= '0' && r <= '9':
			prefix.WriteRune(r)
			seenDigit = true
		case r == '.' && seenDigit:
			prefix.WriteRune(r)
			seenDot = true
		default:
			if !seenDigit {
				return version, nil
			}
			stoppedEarly = true
			goto parse
		}
	}

parse:
	// If we stopped early on a non-version character without ever seeing a dot,
	// the input looks like a git commit hash (e.g. "5f53228") rather than a
	// version number. Return the zero version instead of misinterpreting the
	// leading digit as a major version.
	if stoppedEarly && !seenDot {
		return version, nil
	}
	value = strings.Trim(prefix.String(), ".")
	if value == "" {
		return version, nil
	}

	parts := strings.Split(value, ".")
	for index, part := range parts {
		if index >= len(version) {
			break
		}
		if part == "" {
			continue
		}
		component, err := strconv.ParseUint(part, 10, 16)
		if err != nil && !errors.Is(err, strconv.ErrRange) {
			return version, fmt.Errorf("invalid numeric component %q", part)
		}
		if err != nil {
			component = 0xffff
		}
		version[index] = uint16(component)
	}

	return version, nil
}

func utf16LE(value string, terminate bool) ([]byte, int, error) {
	units := utf16.Encode([]rune(value))
	if terminate {
		units = append(units, 0)
	}
	if len(units) > 0xffff {
		return nil, 0, fmt.Errorf("string %q exceeds 65535 UTF-16 code units", value)
	}

	data := make([]byte, len(units)*2)
	for index, unit := range units {
		binary.LittleEndian.PutUint16(data[index*2:], unit)
	}

	return data, len(units), nil
}

func appendResourceDirectory(dst []byte, idEntries uint16) []byte {
	dst = appendLE32(dst, 0)
	dst = appendLE32(dst, 0)
	dst = appendLE16(dst, 0)
	dst = appendLE16(dst, 0)
	dst = appendLE16(dst, 0)
	dst = appendLE16(dst, idEntries)
	return dst
}

func appendResourceDirectoryEntry(dst []byte, id uint16, offset uint32, directory bool) []byte {
	dst = appendLE32(dst, uint32(id))
	if directory {
		offset |= 0x80000000
	}
	return appendLE32(dst, offset)
}

func appendLE16(dst []byte, value uint16) []byte {
	var scratch [2]byte
	binary.LittleEndian.PutUint16(scratch[:], value)
	return append(dst, scratch[:]...)
}

func appendLE32(dst []byte, value uint32) []byte {
	var scratch [4]byte
	binary.LittleEndian.PutUint32(scratch[:], value)
	return append(dst, scratch[:]...)
}

func padTo4(dst []byte) []byte {
	for len(dst)%4 != 0 {
		dst = append(dst, 0)
	}
	return dst
}

func coffName(value string) [8]byte {
	var name [8]byte
	copy(name[:], value)
	return name
}

func fixedVersionWord(major, minor uint16) uint32 {
	return uint32(major)<<16 | uint32(minor)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

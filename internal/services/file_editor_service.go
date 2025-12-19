package services

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/omnihance/omnihance-a3-agent/internal/logger"
)

type FileType string

const (
	FileTypeNPC     FileType = "a3_npc_file"
	FileTypeDrop    FileType = "a3_drop_file"
	FileTypeMap     FileType = "a3_map_file"
	FileTypeUnknown FileType = "a3_unknown_file"
	FileTypeSpawn   FileType = "a3_spawn_file"
	FileTypeQuest   FileType = "a3_quest_file"
	FileTypeText    FileType = "text_file"
)

const (
	NPCFileSize   = 78
	QuestFileSize = 798
)

const (
	DropFileExtension  = ".itm"
	MapFileExtension   = ".map"
	SpawnFileExtension = ".n_ndt"
	QuestFileExtension = ".dat"
)

type FileEditorService interface {
	IsFileEditable(path string, fileInfo fs.FileInfo) bool
	GetFileType(path string, fileInfo fs.FileInfo) FileType
	GetFileAPIEndpoint(path string, fileInfo fs.FileInfo) string
	IsFileViewable(path string, fileInfo fs.FileInfo) bool
	ReadNPCFileData(path string) (*NPCFileData, error)
	WriteNPCFileData(path string, data *NPCFileData) error
	WriteTextFileData(path string, content string) error
	ReadSpawnFileData(path string) ([]NPCSpawnData, error)
	WriteSpawnFileData(path string, data []NPCSpawnData) error
	Stat(name string) (fs.FileInfo, error)
	ReadDir(name string) ([]fs.DirEntry, error)
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
	Remove(name string) error
	RemoveAll(path string) error
	OpenFile(name string, flag int, perm fs.FileMode) (*os.File, error)
	Hostname() (string, error)
	IsNotExist(err error) bool
	IsExist(err error) bool
	ReadClientMonsterFileData(path string) ([]MonsterClientData, error)
	ReadClientMonsterFileBytes(data []byte) ([]MonsterClientData, error)
	ReadClientMapFileData(path string) ([]MapClientData, error)
	ReadClientMapFileBytes(data []byte) ([]MapClientData, error)
}

type fileEditorService struct {
	logger logger.Logger
}

func NewFileEditorService(logger logger.Logger) FileEditorService {
	return &fileEditorService{logger: logger}
}

func (fes *fileEditorService) IsFileEditable(path string, fileInfo fs.FileInfo) bool {
	switch fes.GetFileType(path, fileInfo) {
	case FileTypeNPC:
		return true
	case FileTypeText:
		return true
	case FileTypeSpawn:
		return true
	default:
		return false
	}
}

func (fes *fileEditorService) GetFileType(path string, fileInfo fs.FileInfo) FileType {
	extension := filepath.Ext(path)
	switch strings.ToLower(extension) {
	case DropFileExtension:
		return FileTypeDrop
	case MapFileExtension:
		return FileTypeMap
	case SpawnFileExtension:
		return FileTypeSpawn
	default:
		if fileInfo.Size() == NPCFileSize {
			return FileTypeNPC
		}

		if fileInfo.Size() == QuestFileSize && extension == QuestFileExtension {
			return FileTypeQuest
		}

		mimeType := mime.TypeByExtension(extension)
		if strings.HasPrefix(mimeType, "text/") || mimeType == "application/json" {
			return FileTypeText
		}
	}

	return FileTypeUnknown
}

func (fes *fileEditorService) IsFileViewable(path string, fileInfo fs.FileInfo) bool {
	switch fes.GetFileType(path, fileInfo) {
	case FileTypeNPC:
		return true
	case FileTypeText:
		return true
	case FileTypeSpawn:
		return true
	default:
		return false
	}
}

func (fes *fileEditorService) GetFileAPIEndpoint(path string, fileInfo fs.FileInfo) string {
	switch fes.GetFileType(path, fileInfo) {
	case FileTypeNPC:
		return "/file-tree/npc-file"
	case FileTypeText:
		return "/file-tree/text-file"
	case FileTypeSpawn:
		return "/file-tree/spawn-file"
	default:
		return ""
	}
}

func (fes *fileEditorService) ReadNPCFileData(path string) (*NPCFileData, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fes.logger.Error("Failed to close file", logger.Field{Key: "error", Value: closeErr})
		}
	}()

	var npcData NPCFileData
	if err := binary.Read(file, binary.LittleEndian, &npcData); err != nil {
		return nil, err
	}

	return &npcData, nil
}

func (fes *fileEditorService) WriteNPCFileData(path string, data *NPCFileData) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}

	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fes.logger.Error("Failed to close file", logger.Field{Key: "error", Value: closeErr})
		}
	}()

	if err := binary.Write(file, binary.LittleEndian, data); err != nil {
		return err
	}

	return nil
}

func (fes *fileEditorService) WriteTextFileData(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

func (fes *fileEditorService) ReadSpawnFileData(path string) ([]NPCSpawnData, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fes.logger.Error("Failed to close file", logger.Field{Key: "error", Value: closeErr})
		}
	}()

	spawnFileStat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	totalSpawns := spawnFileStat.Size() / 8
	spawnData := make([]NPCSpawnData, totalSpawns)
	for i := range spawnData {
		spawnData[i] = NPCSpawnData{}
		err = binary.Read(file, binary.LittleEndian, &spawnData[i])
		if err != nil {
			return nil, err
		}
	}

	return spawnData, nil
}

func (fes *fileEditorService) WriteSpawnFileData(path string, data []NPCSpawnData) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}

	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fes.logger.Error("Failed to close file", logger.Field{Key: "error", Value: closeErr})
		}
	}()

	if err := binary.Write(file, binary.LittleEndian, data); err != nil {
		return err
	}

	return nil
}

func (fes *fileEditorService) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

func (fes *fileEditorService) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(name)
}

func (fes *fileEditorService) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func (fes *fileEditorService) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(name, data, perm)
}

func (fes *fileEditorService) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (fes *fileEditorService) Remove(name string) error {
	return os.Remove(name)
}

func (fes *fileEditorService) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

func (fes *fileEditorService) OpenFile(name string, flag int, perm fs.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}

func (fes *fileEditorService) Hostname() (string, error) {
	return os.Hostname()
}

func (fes *fileEditorService) IsNotExist(err error) bool {
	return os.IsNotExist(err)
}

func (fes *fileEditorService) IsExist(err error) bool {
	return os.IsExist(err)
}

func (fes *fileEditorService) ReadClientMonsterFileData(path string) ([]MonsterClientData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return fes.ReadClientMonsterFileBytes(data)
}

func (fes *fileEditorService) ReadClientMonsterFileBytes(data []byte) ([]MonsterClientData, error) {
	const entrySize = 96
	entryCount := binary.LittleEndian.Uint32(data[:4])
	if len(data) < int(entryCount*entrySize+4) {
		return nil, errors.New("data is too small")
	}

	reader := bytes.NewReader(data[4:])
	monsterData := make([]MonsterClientData, entryCount)
	for i := range monsterData {
		err := binary.Read(reader, binary.LittleEndian, &monsterData[i])
		if err != nil {
			return nil, err
		}
	}

	return monsterData, nil
}

func (fes *fileEditorService) ReadClientMapFileData(path string) ([]MapClientData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return fes.ReadClientMapFileBytes(data)
}

func (fes *fileEditorService) ReadClientMapFileBytes(data []byte) ([]MapClientData, error) {
	const entrySize = 56
	entryCount := binary.LittleEndian.Uint32(data[:4])
	if len(data) < int(entryCount*entrySize+4) {
		return nil, errors.New("data is too small")
	}

	reader := bytes.NewReader(data[4:])
	mapData := make([]MapClientData, entryCount)
	for i := range mapData {
		err := binary.Read(reader, binary.LittleEndian, &mapData[i])
		if err != nil {
			return nil, err
		}
	}

	return mapData, nil
}

func (fes *fileEditorService) ReadQuestFileData(path string) (*QuestFileData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return fes.ReadQuestFileBytes(data)
}

func (fes *fileEditorService) ReadQuestFileBytes(data []byte) (*QuestFileData, error) {
	r := bytes.NewReader(data)

	var quest QuestFileData
	if err := binary.Read(r, binary.LittleEndian, &quest.Header); err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}

	quest.Objectives = make([]Objective, 0, 7)
	for i := range quest.Objectives {
		var block ObjectiveBlock
		if err := binary.Read(r, binary.LittleEndian, &block); err != nil {
			return nil, fmt.Errorf("read objective block %d: %w", i, err)
		}

		obj := Objective{
			Block: block,
		}

		nameLen := block.NameLength()
		if nameLen > 0 {
			name := make([]byte, nameLen)
			if _, err := io.ReadFull(r, name); err != nil {
				return nil, fmt.Errorf("read objective name %d: %w", i, err)
			}

			obj.Name = string(name)
		}

		quest.Objectives = append(quest.Objectives, obj)
	}

	if err := binary.Read(r, binary.LittleEndian, &quest.Continuations); err != nil {
		return nil, fmt.Errorf("read continuation quests: %w", err)
	}

	return &quest, nil
}

type NPCFileData struct {
	Name                [0x14]byte     `json:"name"`
	Id                  uint16         `json:"id"`
	RespawnRate         uint16         `json:"respawn_rate"`
	AttackTypeInfo      byte           `json:"attack_type_info"`
	TargetSelectionInfo byte           `json:"target_selection_info"`
	Defense             byte           `json:"defense"`
	AdditionalDefense   byte           `json:"additional_defense"`
	Attacks             [0x3]NPCAttack `json:"attacks"`
	AttackSpeedLow      uint16         `json:"attack_speed_low"`
	AttackSpeedHigh     uint16         `json:"attack_speed_high"`
	MovementSpeed       uint32         `json:"movement_speed"`
	Level               byte           `json:"level"`
	PlayerExp           uint16         `json:"player_exp"`
	Appearance          byte           `json:"appearance"`
	HP                  uint32         `json:"hp"`
	BlueAttackDefense   uint16         `json:"blue_attack_defense"`
	RedAttackDefense    uint16         `json:"red_attack_defense"`
	GreyAttackDefense   uint16         `json:"grey_attack_defense"`
	MercenaryExp        uint16         `json:"mercenary_exp"`
	Unknown             uint16         `json:"unknown"`
}

type NPCAttack struct {
	Range            uint16 `json:"range"`
	Area             uint16 `json:"area"`
	Damage           uint16 `json:"damage"`
	AdditionalDamage uint16 `json:"additional_damage"`
}

type NPCSpawnData struct {
	Id          uint16
	X           byte
	Y           byte
	Unknown1    uint16
	Orientation byte
	SpwanStep   byte
}

type MonsterClientData struct {
	ID      uint32
	Name    [0x1F]byte
	Unknown [0x3D]byte
}

type MapClientData struct {
	ID       uint32
	Unknown1 uint32
	Unknown2 uint32
	Unknown3 uint32
	Unknown4 uint32
	Unknown5 uint32
	Name     [0x20]byte
}

type QuestFileData struct {
	Header        QuestHeader
	Objectives    []Objective // exactly 7, parsed sequentially
	Continuations [0x3]uint32
}

type QuestHeader struct {
	QuestID         uint16
	QuestIDPadding  uint16
	GiverNPC        uint16
	GiverNPCPadding uint16

	TargetNPCRaw [24]byte // UInt16 + 22 bytes unknown/associated data

	MinLevelRaw uint32 // UInt8 + 3 padding bytes
	MaxLevelRaw uint32 // UInt8 + 3 padding bytes

	Flags uint32

	// Reward item codes (4 slots, last unused)
	RewardItemRaw [4]uint32 // each: UInt16 item code + 2 padding

	// Padding between item codes and counts (bytes 60–67)
	RewardPadding [8]byte

	// Reward item counts (4 slots, last unused)
	RewardCountRaw [4]uint32 // each: UInt8 count + 3 padding

	ExpReward   uint32
	WoonzReward uint32
	LoreReward  uint32

	TailPadding uint32
}

func (h *QuestHeader) RewardItem(i int) uint16 {
	return uint16(h.RewardItemRaw[i])
}

func (h *QuestHeader) RewardCount(i int) uint8 {
	return uint8(h.RewardCountRaw[i])
}

type Objective struct {
	Block ObjectiveBlock
	Name  string // only if NameLength > 0
}

type ObjectiveBlock struct {
	TypeRaw uint32 // UInt8 + 3 padding

	MapIDRaw      uint32 // UInt16 + 2 padding
	LocationIDRaw uint32 // UInt16 + 2 padding

	RadiusRaw uint32 // UInt8 + 3 padding

	TargetIDRaw  uint32 // Monster or NPC ID (UInt16 + padding)
	KillCountRaw uint32 // UInt16 + padding

	QuestItemIDRaw uint32

	DropItem1Raw uint32
	DropItem2Raw uint32
	DropItem3Raw uint32

	Pad1 [16]byte

	RequiredItemCountRaw uint32 // UInt16 + padding

	Pad2 [16]byte

	DropProb1Raw uint32 // UInt8 + padding
	DropProb2Raw uint32
	DropProb3Raw uint32

	Pad3 [4]byte

	NameLengthRaw uint32 // UInt8 + 3 padding
}

func (o *ObjectiveBlock) Type() uint8 {
	return uint8(o.TypeRaw)
}

func (o *ObjectiveBlock) NameLength() uint8 {
	return uint8(o.NameLengthRaw)
}

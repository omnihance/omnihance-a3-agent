package services

import (
	"bytes"
	"encoding/binary"
	"io"
	"io/fs"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"github.com/project-agonyl/agonyl-utils-go/dropfile"
	"github.com/project-agonyl/agonyl-utils-go/itemcombinationdata"
	"github.com/project-agonyl/agonyl-utils-go/itemfile"
	"github.com/project-agonyl/agonyl-utils-go/questfile"
)

type FileType string

const (
	FileTypeNPC                 FileType = "a3_npc_file"
	FileTypeDrop                FileType = "a3_drop_file"
	FileTypeItemCombinationData FileType = "a3_item_combination_data_file"
	FileTypeMap                 FileType = "a3_map_file"
	FileTypeUnknown             FileType = "a3_unknown_file"
	FileTypeSpawn               FileType = "a3_spawn_file"
	FileTypeQuest               FileType = "a3_quest_file"
	FileTypeText                FileType = "text_file"
)

const (
	NPCFileSize = 78
)

const clientDataHeaderSize = 4

const (
	DropFileExtension           = ".itm"
	ItemCombinationDataFileName = "ItemCombinationData"
	MapFileExtension            = ".map"
	SpawnFileExtension          = ".n_ndt"
	QuestFileExtension          = ".dat"
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
	ReadDropFileData(path string) (dropfile.DropFile, error)
	WriteDropFileData(path string, data dropfile.DropFile) error
	ReadItemCombinationData(path string) (itemcombinationdata.ItemCombinationData, error)
	WriteItemCombinationData(path string, data itemcombinationdata.ItemCombinationData) error
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
	ReadClientIT0FileBytes(data []byte) ([]itemfile.Item, error)
	ReadClientIT1FileBytes(data []byte) ([]itemfile.Item, error)
	ReadClientIT2FileBytes(data []byte) ([]itemfile.Item, error)
	ReadClientIT3FileBytes(data []byte) ([]itemfile.Item, error)
	ReadQuestFileData(path string) (questfile.QuestFile, error)
	WriteQuestFileData(path string, data questfile.QuestFile) error
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
	case FileTypeQuest:
		return true
	case FileTypeDrop:
		return true
	case FileTypeItemCombinationData:
		return true
	default:
		return false
	}
}

func (fes *fileEditorService) GetFileType(path string, fileInfo fs.FileInfo) FileType {
	if strings.EqualFold(filepath.Base(path), ItemCombinationDataFileName) {
		return FileTypeItemCombinationData
	}

	extension := filepath.Ext(path)
	switch strings.ToLower(extension) {
	case DropFileExtension:
		return FileTypeDrop
	case MapFileExtension:
		return FileTypeMap
	case SpawnFileExtension:
		return FileTypeSpawn
	case QuestFileExtension:
		return FileTypeQuest
	default:
		if fileInfo.Size() == NPCFileSize {
			return FileTypeNPC
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
	case FileTypeQuest:
		return true
	case FileTypeDrop:
		return true
	case FileTypeItemCombinationData:
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
	case FileTypeQuest:
		return "/file-tree/quest-file"
	case FileTypeDrop:
		return "/file-tree/drop-file"
	case FileTypeItemCombinationData:
		return "/file-tree/item-combination-data"
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

func (fes *fileEditorService) ReadDropFileData(path string) (dropfile.DropFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fes.logger.Error("Failed to close file", logger.Field{Key: "error", Value: closeErr})
		}
	}()

	return dropfile.Read(file)
}

func (fes *fileEditorService) WriteDropFileData(path string, data dropfile.DropFile) error {
	var buf bytes.Buffer
	if err := dropfile.Write(&buf, data); err != nil {
		return err
	}

	_, err := dropfile.Read(bytes.NewReader(buf.Bytes()))
	if err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}

	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fes.logger.Error("Failed to close file", logger.Field{Key: "error", Value: closeErr})
		}
	}()

	_, err = file.Write(buf.Bytes())
	return err
}

func (fes *fileEditorService) ReadItemCombinationData(path string) (itemcombinationdata.ItemCombinationData, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fes.logger.Error("Failed to close file", logger.Field{Key: "error", Value: closeErr})
		}
	}()

	return itemcombinationdata.Read(file)
}

func (fes *fileEditorService) WriteItemCombinationData(path string, data itemcombinationdata.ItemCombinationData) error {
	var buf bytes.Buffer
	if err := itemcombinationdata.Write(&buf, data); err != nil {
		return err
	}

	_, err := itemcombinationdata.Read(bytes.NewReader(buf.Bytes()))
	if err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}

	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fes.logger.Error("Failed to close file", logger.Field{Key: "error", Value: closeErr})
		}
	}()

	_, err = file.Write(buf.Bytes())
	return err
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
	entryCount, err := readClientDataEntryCount(data, entrySize)
	if err != nil {
		return nil, err
	}

	reader := bytes.NewReader(data[clientDataHeaderSize:])
	monsterData := make([]MonsterClientData, int(entryCount))
	for i := range monsterData {
		err = binary.Read(reader, binary.LittleEndian, &monsterData[i])
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
	entryCount, err := readClientDataEntryCount(data, entrySize)
	if err != nil {
		return nil, err
	}

	reader := bytes.NewReader(data[clientDataHeaderSize:])
	mapData := make([]MapClientData, int(entryCount))
	for i := range mapData {
		err = binary.Read(reader, binary.LittleEndian, &mapData[i])
		if err != nil {
			return nil, err
		}
	}

	return mapData, nil
}

func readClientDataEntryCount(data []byte, entrySize uint64) (uint32, error) {
	if len(data) < clientDataHeaderSize {
		return 0, io.ErrUnexpectedEOF
	}

	entryCount := binary.LittleEndian.Uint32(data[:clientDataHeaderSize])
	requiredSize := uint64(clientDataHeaderSize) + uint64(entryCount)*entrySize
	if requiredSize > uint64(len(data)) {
		return 0, io.ErrUnexpectedEOF
	}

	return entryCount, nil
}

func (fes *fileEditorService) ReadClientIT0FileBytes(data []byte) ([]itemfile.Item, error) {
	rawItems, err := itemfile.ReadIT0(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	return itemfile.ParseIT0Items(rawItems, nil)
}

func (fes *fileEditorService) ReadClientIT1FileBytes(data []byte) ([]itemfile.Item, error) {
	return itemfile.ReadIT1Items(bytes.NewReader(data))
}

func (fes *fileEditorService) ReadClientIT2FileBytes(data []byte) ([]itemfile.Item, error) {
	return itemfile.ReadIT2Items(bytes.NewReader(data))
}

func (fes *fileEditorService) ReadClientIT3FileBytes(data []byte) ([]itemfile.Item, error) {
	return itemfile.ReadIT3Items(bytes.NewReader(data))
}

func (fes *fileEditorService) ReadQuestFileData(path string) (questfile.QuestFile, error) {
	questFile, err := os.Open(path)
	if err != nil {
		return questfile.QuestFile{}, err
	}

	defer func() {
		if closeErr := questFile.Close(); closeErr != nil {
			fes.logger.Error("Failed to close file", logger.Field{Key: "error", Value: closeErr})
		}
	}()

	return questfile.Read(questFile)
}

func (fes *fileEditorService) WriteQuestFileData(path string, data questfile.QuestFile) error {
	var buf bytes.Buffer
	if err := questfile.Write(&buf, data); err != nil {
		return err
	}

	_, err := questfile.Read(bytes.NewReader(buf.Bytes()))
	if err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}

	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fes.logger.Error("Failed to close file", logger.Field{Key: "error", Value: closeErr})
		}
	}()

	_, err = file.Write(buf.Bytes())
	return err
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

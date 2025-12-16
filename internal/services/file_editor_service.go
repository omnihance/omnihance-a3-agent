package services

import (
	"encoding/binary"
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
	FileTypeText    FileType = "text_file"
)

const (
	NPCFileSize = 78
)

const (
	DropFileExtension  = ".itm"
	MapFileExtension   = ".map"
	SpawnFileExtension = ".n_ndt"
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

		mimeType := mime.TypeByExtension(extension)
		if strings.HasPrefix(mimeType, "text/") {
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

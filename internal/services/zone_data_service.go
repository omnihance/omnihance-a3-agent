package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/omnihance/omnihance-a3-agent/internal/db"
	"github.com/omnihance/omnihance-a3-agent/internal/utils"
)

type ZoneDataFormat string

const (
	ZoneDataFormatMap               ZoneDataFormat = "zone_map"
	ZoneDataFormatNPCSkill          ZoneDataFormat = "npc_skill"
	ZoneDataFormatNPCFavor          ZoneDataFormat = "npc_favor"
	ZoneDataFormatPCData            ZoneDataFormat = "pc_data"
	ZoneDataFormatSkillData         ZoneDataFormat = "skill_data"
	ZoneDataFormatSkillDelay        ZoneDataFormat = "skill_delay"
	ZoneDataFormatPassiveSkill      ZoneDataFormat = "passive_skill"
	ZoneDataFormatHiredSoldierSkill ZoneDataFormat = "hired_soldier_skill"
	ZoneDataFormatCashItem          ZoneDataFormat = "cash_item"
	ZoneDataFormatSetItem           ZoneDataFormat = "set_item"
	ZoneDataFormatPresentItemSet    ZoneDataFormat = "present_item_set"
	ZoneDataFormatPet               ZoneDataFormat = "pet"
	ZoneDataFormatShueCombination   ZoneDataFormat = "shue_combination"
	ZoneDataFormatLottery           ZoneDataFormat = "lottery"
	ZoneDataFormatDerbyGift         ZoneDataFormat = "derby_gift"
	ZoneDataFormatEventItemReward   ZoneDataFormat = "event_item_reward"
	ZoneDataFormatA3Present         ZoneDataFormat = "a3_present"
	ZoneDataFormatMessage           ZoneDataFormat = "message"
	ZoneDataFormatQuestEx           ZoneDataFormat = "quest_ex"
	ZoneDataFormatSQuestQuiz        ZoneDataFormat = "squest_quiz"
	ZoneDataFormatTowerTreasure     ZoneDataFormat = "tower_treasure"
	ZoneDataFormatOXQuiz            ZoneDataFormat = "ox_quiz"
	ZoneDataFormatTyrBase           ZoneDataFormat = "tyr_base"
	ZoneDataFormatTyrPortal         ZoneDataFormat = "tyr_portal"
	ZoneDataFormatTyrUpgrade        ZoneDataFormat = "tyr_upgrade"
	ZoneDataFormatTyrStartPoint     ZoneDataFormat = "tyr_start_point"
	ZoneDataFormatTyrGift           ZoneDataFormat = "tyr_gift"
	ZoneDataFormatTyrNPCRegen       ZoneDataFormat = "tyr_npc_regen"
	ZoneDataFormatTyrSkillLayer     ZoneDataFormat = "tyr_skill_layer"
)

var ErrZoneDataStale = errors.New("zone data source hash is stale")

type ZoneDataField struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Type     string `json:"type"`
	Scope    string `json:"scope"`
	Editable bool   `json:"editable"`
	Min      *int64 `json:"min,omitempty"`
	Max      *int64 `json:"max,omitempty"`
}

type ZoneDataRow struct {
	Index       int            `json:"index"`
	Values      map[string]any `json:"values"`
	OpaqueBytes string         `json:"opaque_bytes"`
}

type ZoneDataMap struct {
	Name     string        `json:"name"`
	Warps    []ZoneDataRow `json:"warps"`
	Cells    []uint32      `json:"cells"`
	Width    int           `json:"width"`
	Height   int           `json:"height"`
	Trailing string        `json:"trailing_opaque_bytes"`
}

type ZoneDataCapabilities struct {
	UpdateFields bool `json:"update_fields"`
	InsertRows   bool `json:"insert_rows"`
	DeleteRows   bool `json:"delete_rows"`
}

type ZoneDataFile struct {
	Format       ZoneDataFormat       `json:"format"`
	SourceHash   string               `json:"source_hash"`
	Schema       []ZoneDataField      `json:"schema"`
	Rows         []ZoneDataRow        `json:"rows,omitempty"`
	Map          *ZoneDataMap         `json:"map,omitempty"`
	Capabilities ZoneDataCapabilities `json:"capabilities"`
}

type ZoneDataOperation struct {
	Scope string          `json:"scope"`
	Row   int             `json:"row"`
	Field string          `json:"field"`
	Value json.RawMessage `json:"value"`
}

type ZoneDataService interface {
	ResolveRoot() (string, error)
	Detect(root string, path string) (ZoneDataFormat, bool)
	DetectResolved(root string, path string) (ZoneDataFormat, bool)
	Read(path string, format ZoneDataFormat) (ZoneDataFile, error)
	Apply(original []byte, format ZoneDataFormat, operations []ZoneDataOperation) ([]byte, error)
}

type zoneDataService struct {
	internalDB db.InternalDB
	fileEditor FileEditorService
}

func NewZoneDataService(internalDB db.InternalDB, fileEditor FileEditorService) ZoneDataService {
	return &zoneDataService{internalDB: internalDB, fileEditor: fileEditor}
}

func (s *zoneDataService) ResolveRoot() (string, error) {
	setting, err := s.internalDB.GetSetting(constants.SettingKeyZoneServerPath)
	if err != nil {
		return "", err
	}

	root := strings.TrimSpace(setting.Value)
	if root == "" {
		return "", fmt.Errorf("zone server path is not configured")
	}

	rootPath, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}

	return filepath.EvalSymlinks(filepath.Clean(rootPath))
}

func (s *zoneDataService) Detect(root string, path string) (ZoneDataFormat, bool) {
	rootPath, err := filepath.EvalSymlinks(filepath.Clean(root))
	if err != nil {
		return "", false
	}

	return s.DetectResolved(rootPath, path)
}

func (s *zoneDataService) DetectResolved(root string, path string) (ZoneDataFormat, bool) {
	if !IsZoneDataCandidatePath(path) {
		return "", false
	}

	rootPath := filepath.Clean(root)
	filePath, err := filepath.EvalSymlinks(filepath.Clean(path))
	if err != nil {
		return "", false
	}

	relative, err := filepath.Rel(rootPath, filePath)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", false
	}

	parts := strings.Split(strings.ToLower(filepath.ToSlash(relative)), "/")
	if len(parts) == 1 && parts[0] == "a3msg_zone_tw.dat" {
		return ZoneDataFormatMessage, true
	}
	if len(parts) == 1 && (numberedDataFile(parts[0], "present_") || numberedDataFile(parts[0], "reload_present_")) {
		return ZoneDataFormatA3Present, true
	}

	if len(parts) == 2 && parts[0] == "event" {
		if parts[1] == "lotteryitem.dat" {
			return ZoneDataFormatLottery, true
		}

		if strings.HasPrefix(parts[1], "eventitem") && strings.HasSuffix(parts[1], ".dat") {
			return ZoneDataFormatEventItemReward, true
		}
	}
	if len(parts) == 2 && parts[0] == "tower" && numberedExtensionFile(parts[1], ".itm", 6) {
		return ZoneDataFormatTowerTreasure, true
	}
	if len(parts) == 2 && parts[0] == "oxquiz" && parts[1] == "oxquiztable.dat" {
		return ZoneDataFormatOXQuiz, true
	}
	if len(parts) == 3 && parts[0] == "zonedata" && parts[1] == "quest" && fixedDigitDataFile(parts[2], 4) {
		return ZoneDataFormatQuestEx, true
	}
	if len(parts) == 3 && parts[0] == "zonedata" && parts[1] == "squest" && parts[2] == "quiztable.dat" {
		return ZoneDataFormatSQuestQuiz, true
	}
	if len(parts) == 3 && parts[0] == "zonedata" && parts[1] == "tyr" {
		switch parts[2] {
		case "baseinfo.tyr":
			return ZoneDataFormatTyrBase, true
		case "warpportal.tyr":
			return ZoneDataFormatTyrPortal, true
		case "upgrade.tyr":
			return ZoneDataFormatTyrUpgrade, true
		case "startpoint.tyr":
			return ZoneDataFormatTyrStartPoint, true
		case "tyrgift.dat":
			return ZoneDataFormatTyrGift, true
		case "npcregen.tyr":
			return ZoneDataFormatTyrNPCRegen, true
		case "skilllayer.tyr":
			return ZoneDataFormatTyrSkillLayer, true
		}
	}

	if len(parts) == 3 && parts[0] == "zonedata" {
		switch parts[1] {
		case "map":
			if strings.HasSuffix(parts[2], ".map") {
				return ZoneDataFormatMap, true
			}
		case "npc":
			switch parts[2] {
			case "npcskill":
				return ZoneDataFormatNPCSkill, true
			case "favindex.dat":
				return ZoneDataFormatNPCFavor, true
			case "derbygift.dat":
				return ZoneDataFormatDerbyGift, true
			}
		case "shop":
			if parts[2] == "cashitemtbl.dat" {
				return ZoneDataFormatCashItem, true
			}
		case "item":
			switch parts[2] {
			case "sit0", "sit1", "sit2", "sit3":
				return ZoneDataFormatSetItem, true
			case "presentitemset.dat":
				return ZoneDataFormatPresentItemSet, true
			case "pet":
				return ZoneDataFormatPet, true
			case "shuecombinationdata":
				return ZoneDataFormatShueCombination, true
			}
		case "pc":
			if isClassFile(parts[2]) {
				return ZoneDataFormatPCData, true
			}
		case "skill":
			if isClassFile(parts[2]) {
				return ZoneDataFormatSkillData, true
			}
			switch parts[2] {
			case "skilldelay.dat":
				return ZoneDataFormatSkillDelay, true
			case "psvskill.dat":
				return ZoneDataFormatPassiveSkill, true
			case "hsst0", "hsst1", "hsst2", "hsst3":
				return ZoneDataFormatHiredSoldierSkill, true
			}
		}
	}

	return "", false
}

func IsZoneDataCandidatePath(path string) bool {
	extension := strings.ToLower(filepath.Ext(path))
	switch extension {
	case ".dat", ".itm", ".map", ".tyr":
		return true
	case "":
		name := strings.ToLower(filepath.Base(path))
		switch name {
		case "0", "1", "2", "3", "npcskill", "pet", "shuecombinationdata",
			"sit0", "sit1", "sit2", "sit3", "hsst0", "hsst1", "hsst2", "hsst3":
			return true
		}
	}

	return false
}

func (s *zoneDataService) Read(path string, format ZoneDataFormat) (ZoneDataFile, error) {
	data, err := s.fileEditor.ReadFile(path)
	if err != nil {
		return ZoneDataFile{}, err
	}

	decoded, err := decodeZoneData(data, format)
	if err != nil {
		return ZoneDataFile{}, err
	}

	decoded.SourceHash = utils.CalculateFileHash(data)
	return decoded, nil
}

func isClassFile(value string) bool {
	return value == "0" || value == "1" || value == "2" || value == "3"
}

func numberedDataFile(value string, prefix string) bool {
	if !strings.HasPrefix(value, prefix) || !strings.HasSuffix(value, ".dat") {
		return false
	}

	number := strings.TrimSuffix(strings.TrimPrefix(value, prefix), ".dat")
	_, err := strconv.ParseUint(number, 10, 32)
	return err == nil
}

func fixedDigitDataFile(value string, digits int) bool {
	if len(value) != digits+4 || !strings.HasSuffix(value, ".dat") {
		return false
	}

	_, err := strconv.ParseUint(value[:digits], 10, 32)
	return err == nil
}

func numberedExtensionFile(value string, extension string, max uint64) bool {
	if !strings.HasSuffix(value, extension) {
		return false
	}

	parsed, err := strconv.ParseUint(strings.TrimSuffix(value, extension), 10, 32)
	return err == nil && parsed <= max
}

func decodeZoneData(data []byte, format ZoneDataFormat) (ZoneDataFile, error) {
	result := ZoneDataFile{
		Format:       format,
		Capabilities: ZoneDataCapabilities{UpdateFields: true},
	}

	var err error
	switch format {
	case ZoneDataFormatMap:
		result.Schema = mapSchema()
		result.Map, err = decodeMap(data)
	case ZoneDataFormatNPCSkill:
		result.Schema = npcSkillSchema()
		result.Rows, err = decodeNPCSkill(data)
	case ZoneDataFormatNPCFavor:
		result.Schema = npcFavorSchema()
		result.Rows, err = decodeNPCFavor(data)
	case ZoneDataFormatPCData:
		result.Schema = pcDataSchema()
		result.Rows, err = decodePCData(data)
	case ZoneDataFormatSkillData:
		result.Schema = skillDataSchema()
		result.Rows, err = decodeSkillData(data)
	case ZoneDataFormatSkillDelay:
		result.Schema = skillDelaySchema()
		result.Rows, err = decodeSkillDelay(data)
	case ZoneDataFormatPassiveSkill:
		result.Schema = passiveSkillSchema()
		result.Rows, err = decodePassiveSkill(data)
	case ZoneDataFormatHiredSoldierSkill:
		result.Schema = hiredSoldierSkillSchema()
		result.Rows, err = decodeHiredSoldierSkill(data)
	case ZoneDataFormatCashItem, ZoneDataFormatSetItem, ZoneDataFormatPresentItemSet, ZoneDataFormatPet,
		ZoneDataFormatShueCombination, ZoneDataFormatLottery, ZoneDataFormatDerbyGift,
		ZoneDataFormatEventItemReward, ZoneDataFormatA3Present:
		result.Schema, result.Rows, err = decodeEconomyZoneData(data, format)
	case ZoneDataFormatMessage, ZoneDataFormatQuestEx, ZoneDataFormatSQuestQuiz, ZoneDataFormatTowerTreasure,
		ZoneDataFormatOXQuiz, ZoneDataFormatTyrBase, ZoneDataFormatTyrPortal, ZoneDataFormatTyrUpgrade,
		ZoneDataFormatTyrStartPoint, ZoneDataFormatTyrGift, ZoneDataFormatTyrNPCRegen, ZoneDataFormatTyrSkillLayer:
		result.Schema, result.Rows, err = decodeRemainingZoneData(data, format)
	default:
		err = fmt.Errorf("unsupported ZoneData format %q", format)
	}

	return result, err
}

func row(index int, raw []byte, values map[string]any) ZoneDataRow {
	return ZoneDataRow{Index: index, Values: values, OpaqueBytes: base64.StdEncoding.EncodeToString(raw)}
}

func integerField(key string, label string, scope string, max int64) ZoneDataField {
	min := int64(0)
	return ZoneDataField{Key: key, Label: label, Type: "integer", Scope: scope, Editable: true, Min: &min, Max: &max}
}

func boolField(key string, label string, scope string) ZoneDataField {
	return ZoneDataField{Key: key, Label: label, Type: "boolean", Scope: scope, Editable: true}
}

func stringField(key string, label string, scope string) ZoneDataField {
	return ZoneDataField{Key: key, Label: label, Type: "string", Scope: scope, Editable: true}
}

func writeZoneData(write func(io.Writer) error) ([]byte, error) {
	var buffer bytes.Buffer
	if err := write(&buffer); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func operationUint(operation ZoneDataOperation, max uint64) (uint64, error) {
	var value uint64
	if err := json.Unmarshal(operation.Value, &value); err != nil {
		return 0, fmt.Errorf("%s must be an integer", operation.Field)
	}

	if value > max {
		return 0, fmt.Errorf("%s exceeds %d", operation.Field, max)
	}

	return value, nil
}

func operationInt32(operation ZoneDataOperation) (int32, error) {
	var value int64
	if err := json.Unmarshal(operation.Value, &value); err != nil || value < -2147483648 || value > 2147483647 {
		return 0, fmt.Errorf("%s must be a signed 32-bit integer", operation.Field)
	}

	return int32(value), nil
}

func validateRow(row int, length int) error {
	if row < 0 || row >= length {
		return fmt.Errorf("row %d out of range", row)
	}

	return nil
}

func verifyZoneData(original []byte, encoded []byte, allowed []bool, format ZoneDataFormat) error {
	if len(original) != len(encoded) || len(allowed) != len(original) {
		return fmt.Errorf("ZoneData byte length changed unexpectedly")
	}

	for index := range original {
		if original[index] != encoded[index] && !allowed[index] {
			return fmt.Errorf("opaque byte %d changed", index)
		}
	}

	decoded, err := decodeZoneData(encoded, format)
	if err != nil {
		return fmt.Errorf("reparse ZoneData: %w", err)
	}

	if decoded.Format != format {
		return fmt.Errorf("ZoneData format changed unexpectedly")
	}

	return nil
}

func allowBytes(allowed []bool, offset int, size int) {
	for index := offset; index < offset+size && index < len(allowed); index++ {
		allowed[index] = true
	}
}

func fieldIndex(field string, prefix string, count int) (int, bool) {
	if !strings.HasPrefix(field, prefix) {
		return 0, false
	}

	index, err := strconv.Atoi(strings.TrimPrefix(field, prefix))
	return index, err == nil && index >= 0 && index < count
}

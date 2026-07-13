package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/project-agonyl/agonyl-utils-go/hiredsoldierskillfile"
	"github.com/project-agonyl/agonyl-utils-go/npcfavorfile"
	"github.com/project-agonyl/agonyl-utils-go/npcskillfile"
	"github.com/project-agonyl/agonyl-utils-go/passiveskillfile"
	"github.com/project-agonyl/agonyl-utils-go/pcdatafile"
	"github.com/project-agonyl/agonyl-utils-go/skilldatafile"
	"github.com/project-agonyl/agonyl-utils-go/skilldelayfile"
	"github.com/project-agonyl/agonyl-utils-go/zonemapfile"
)

func (s *zoneDataService) Apply(original []byte, format ZoneDataFormat, operations []ZoneDataOperation) ([]byte, error) {
	allowed := make([]bool, len(original))
	var encoded []byte
	var err error
	switch format {
	case ZoneDataFormatMap:
		encoded, err = applyMap(original, operations, allowed)
	case ZoneDataFormatNPCSkill:
		encoded, err = applyNPCSkill(original, operations, allowed)
	case ZoneDataFormatNPCFavor:
		encoded, err = applyNPCFavor(original, operations, allowed)
	case ZoneDataFormatPCData:
		encoded, err = applyPCData(original, operations, allowed)
	case ZoneDataFormatSkillData:
		encoded, err = applySkillData(original, operations, allowed)
	case ZoneDataFormatSkillDelay:
		encoded, err = applySkillDelay(original, operations, allowed)
	case ZoneDataFormatPassiveSkill:
		encoded, err = applyPassiveSkill(original, operations, allowed)
	case ZoneDataFormatHiredSoldierSkill:
		encoded, err = applyHiredSoldierSkill(original, operations, allowed)
	case ZoneDataFormatCashItem, ZoneDataFormatSetItem, ZoneDataFormatPresentItemSet, ZoneDataFormatPet,
		ZoneDataFormatShueCombination, ZoneDataFormatLottery, ZoneDataFormatDerbyGift,
		ZoneDataFormatEventItemReward, ZoneDataFormatA3Present:
		encoded, err = applyEconomyZoneData(original, format, operations, allowed)
	default:
		err = fmt.Errorf("unsupported ZoneData format %q", format)
	}
	if err != nil {
		return nil, err
	}

	if err := verifyZoneData(original, encoded, allowed, format); err != nil {
		return nil, err
	}

	return encoded, nil
}

func applyMap(original []byte, operations []ZoneDataOperation, allowed []bool) ([]byte, error) {
	data, err := zonemapfile.Read(bytes.NewReader(original))
	if err != nil {
		return nil, err
	}

	meshOffset := zonemapfile.HeaderSize + len(data.Warps)*zonemapfile.WarpSize
	for _, operation := range operations {
		switch operation.Scope {
		case "map":
			if operation.Field != "name" {
				return nil, fmt.Errorf("unknown map field %q", operation.Field)
			}
			var value string
			if err := json.Unmarshal(operation.Value, &value); err != nil {
				return nil, fmt.Errorf("map name must be a string")
			}
			if err := data.SetName(value); err != nil {
				return nil, err
			}
			allowBytes(allowed, 20, 2)
		case "warp":
			if err := validateRow(operation.Row, len(data.Warps)); err != nil {
				return nil, err
			}
			value, err := operationUint(operation, 65535)
			if err != nil {
				return nil, err
			}
			offset := zonemapfile.HeaderSize + operation.Row*zonemapfile.WarpSize
			switch operation.Field {
			case "map_id":
				data.Warps[operation.Row].SetMapID(uint16(value))
				allowBytes(allowed, offset, 2)
			case "cell":
				data.Warps[operation.Row].SetCell(uint16(value))
				allowBytes(allowed, offset+2, 2)
			case "unknown":
				data.Warps[operation.Row].SetUnknown(uint16(value))
				allowBytes(allowed, offset+4, 2)
			default:
				return nil, fmt.Errorf("unknown warp field %q", operation.Field)
			}
		case "cell":
			if err := validateRow(operation.Row, len(data.Cells)); err != nil {
				return nil, err
			}
			raw := data.Cells[operation.Row]
			switch operation.Field {
			case "can_move":
				var value bool
				if err := json.Unmarshal(operation.Value, &value); err != nil {
					return nil, fmt.Errorf("can_move must be a boolean")
				}
				data.Cells[operation.Row] = zonemapfile.SetCanMove(raw, value)
			case "pk_level":
				value, err := operationUint(operation, 3)
				if err != nil {
					return nil, err
				}
				data.Cells[operation.Row], err = zonemapfile.SetPKLevel(raw, byte(value))
				if err != nil {
					return nil, err
				}
			case "warp_index":
				value, err := operationUint(operation, 15)
				if err != nil {
					return nil, err
				}
				if value == 15 {
					data.Cells[operation.Row], err = zonemapfile.SetWarpIndex(raw, nil)
				} else {
					index := byte(value)
					data.Cells[operation.Row], err = zonemapfile.SetWarpIndex(raw, &index)
				}
				if err != nil {
					return nil, err
				}
			default:
				return nil, fmt.Errorf("unknown cell field %q", operation.Field)
			}
			allowBytes(allowed, meshOffset+operation.Row*4, 4)
		default:
			return nil, fmt.Errorf("unknown operation scope %q", operation.Scope)
		}
	}

	return writeZoneData(func(writer io.Writer) error { return zonemapfile.Write(writer, data) })
}

func applyNPCSkill(original []byte, operations []ZoneDataOperation, allowed []bool) ([]byte, error) {
	data, err := npcskillfile.Read(bytes.NewReader(original))
	if err != nil {
		return nil, err
	}
	for _, operation := range operations {
		if operation.Scope != "row" {
			return nil, fmt.Errorf("invalid NPC skill scope %q", operation.Scope)
		}
		if err := validateRow(operation.Row, len(data)); err != nil {
			return nil, err
		}
		value, err := operationUint(operation, 65535)
		if err != nil {
			return nil, err
		}
		offset := operation.Row * npcskillfile.RecordSize
		switch operation.Field {
		case "npc_type":
			data[operation.Row].SetNPCType(uint16(value))
			allowBytes(allowed, offset, 2)
		case "kind":
			data[operation.Row].SetKind(byte(value))
			allowBytes(allowed, offset+2, 1)
		case "attack_type":
			data[operation.Row].SetAttackType(byte(value))
			allowBytes(allowed, offset+3, 1)
		case "one_target_range":
			data[operation.Row].SetOneTargetRange(byte(value))
			allowBytes(allowed, offset+8, 1)
		case "range_radius":
			data[operation.Row].SetRangeRadius(byte(value))
			allowBytes(allowed, offset+9, 1)
		case "cooldown_seconds":
			data[operation.Row].SetCooldownSeconds(uint16(value))
			allowBytes(allowed, offset+10, 2)
		case "effect_param":
			data[operation.Row].SetEffectParam(uint16(value))
			allowBytes(allowed, offset+12, 2)
		case "effect_code":
			data[operation.Row].SetEffectCode(uint16(value))
			allowBytes(allowed, offset+14, 2)
		case "effect_value":
			data[operation.Row].SetEffectValue(uint16(value))
			allowBytes(allowed, offset+16, 2)
		default:
			return nil, fmt.Errorf("unknown NPC skill field %q", operation.Field)
		}
	}
	return writeZoneData(func(writer io.Writer) error { return npcskillfile.Write(writer, data) })
}

func applyNPCFavor(original []byte, operations []ZoneDataOperation, allowed []bool) ([]byte, error) {
	data, err := npcfavorfile.Read(bytes.NewReader(original))
	if err != nil {
		return nil, err
	}
	for _, operation := range operations {
		if operation.Scope != "row" {
			return nil, fmt.Errorf("invalid NPC favor scope %q", operation.Scope)
		}
		if err := validateRow(operation.Row, len(data)); err != nil {
			return nil, err
		}
		offset := operation.Row * npcfavorfile.RecordSize
		switch operation.Field {
		case "index":
			value, err := operationInt32(operation)
			if err != nil {
				return nil, err
			}
			data[operation.Row].SetIndex(value)
			allowBytes(allowed, offset, 4)
		case "npc_type":
			value, err := operationUint(operation, 4294967295)
			if err != nil {
				return nil, err
			}
			data[operation.Row].SetNPCType(uint32(value))
			allowBytes(allowed, offset+4, 4)
		default:
			return nil, fmt.Errorf("unknown NPC favor field %q", operation.Field)
		}
	}
	return writeZoneData(func(writer io.Writer) error { return npcfavorfile.Write(writer, data) })
}

func applyPCData(original []byte, operations []ZoneDataOperation, allowed []bool) ([]byte, error) {
	data, err := pcdatafile.Read(bytes.NewReader(original))
	if err != nil {
		return nil, err
	}
	for _, operation := range operations {
		if operation.Scope != "row" || operation.Row != 0 {
			return nil, fmt.Errorf("invalid PC data target")
		}
		index, ok := fieldIndex(operation.Field, "value_", len(pcDataLabels))
		if !ok {
			return nil, fmt.Errorf("unknown PC data field %q", operation.Field)
		}
		value, err := operationUint(operation, 65535)
		if err != nil {
			return nil, err
		}
		if err := data.SetValue(index, uint16(value)); err != nil {
			return nil, err
		}
		allowBytes(allowed, index*2, 2)
	}
	return writeZoneData(func(writer io.Writer) error { return pcdatafile.Write(writer, data) })
}

func applySkillData(original []byte, operations []ZoneDataOperation, allowed []bool) ([]byte, error) {
	data, err := skilldatafile.Read(bytes.NewReader(original))
	if err != nil {
		return nil, err
	}
	for _, operation := range operations {
		if operation.Scope != "row" {
			return nil, fmt.Errorf("invalid skill data scope %q", operation.Scope)
		}
		if err := validateRow(operation.Row, len(data)); err != nil {
			return nil, err
		}
		value, err := operationUint(operation, 255)
		if err != nil {
			return nil, err
		}
		offset := operation.Row * skilldatafile.RecordSize
		switch operation.Field {
		case "code":
			data[operation.Row].SetCode(byte(value))
			allowBytes(allowed, offset, 1)
		case "type":
			data[operation.Row].SetType(byte(value))
			allowBytes(allowed, offset+1, 1)
		case "sub_type":
			data[operation.Row].SetSubType(byte(value))
			allowBytes(allowed, offset+2, 1)
		case "target_type":
			data[operation.Row].SetTargetType(byte(value))
			allowBytes(allowed, offset+3, 1)
		case "need_item":
			data[operation.Row].SetNeedItem(byte(value))
			allowBytes(allowed, offset+4, 1)
		case "reaction":
			data[operation.Row].SetReaction(byte(value))
			allowBytes(allowed, offset+5, 1)
		case "abnormalcy":
			data[operation.Row].SetAbnormalcy(byte(value))
			allowBytes(allowed, offset+6, 1)
		default:
			index, ok := fieldIndex(operation.Field, "monster_rate_", 6)
			if !ok {
				return nil, fmt.Errorf("unknown skill data field %q", operation.Field)
			}
			if err := data[operation.Row].SetMonsterRate(index, byte(value)); err != nil {
				return nil, err
			}
			allowBytes(allowed, offset+7+index, 1)
		}
	}
	return writeZoneData(func(writer io.Writer) error { return skilldatafile.Write(writer, data) })
}

func applySkillDelay(original []byte, operations []ZoneDataOperation, allowed []bool) ([]byte, error) {
	data, err := skilldelayfile.Read(bytes.NewReader(original))
	if err != nil {
		return nil, err
	}
	for _, operation := range operations {
		if operation.Scope != "row" {
			return nil, fmt.Errorf("invalid skill delay scope %q", operation.Scope)
		}
		if err := validateRow(operation.Row, len(data)); err != nil {
			return nil, err
		}
		offset := operation.Row * skilldelayfile.RecordSize
		switch operation.Field {
		case "class_index":
			value, err := operationUint(operation, 3)
			if err != nil {
				return nil, err
			}
			data[operation.Row].SetClassIndex(byte(value))
			allowBytes(allowed, offset, 1)
		case "skill_index":
			value, err := operationUint(operation, 63)
			if err != nil {
				return nil, err
			}
			data[operation.Row].SetSkillIndex(uint16(value))
			allowBytes(allowed, offset+1, 2)
		case "delay_ms":
			value, err := operationUint(operation, 65535)
			if err != nil {
				return nil, err
			}
			data[operation.Row].SetDelayMS(uint16(value))
			allowBytes(allowed, offset+3, 2)
		case "delay_info":
			value, err := operationUint(operation, 255)
			if err != nil {
				return nil, err
			}
			data[operation.Row].SetDelayInfo(byte(value))
			allowBytes(allowed, offset+4, 1)
		default:
			return nil, fmt.Errorf("unknown skill delay field %q", operation.Field)
		}
	}
	return writeZoneData(func(writer io.Writer) error { return skilldelayfile.Write(writer, data) })
}

func applyPassiveSkill(original []byte, operations []ZoneDataOperation, allowed []bool) ([]byte, error) {
	data, err := passiveskillfile.Read(bytes.NewReader(original))
	if err != nil {
		return nil, err
	}
	for _, operation := range operations {
		if operation.Scope != "row" {
			return nil, fmt.Errorf("invalid passive skill scope %q", operation.Scope)
		}
		if err := validateRow(operation.Row, len(data)); err != nil {
			return nil, err
		}
		offset := operation.Row * passiveskillfile.RecordSize
		value, err := operationUint(operation, 4294967295)
		if err != nil {
			return nil, err
		}
		switch operation.Field {
		case "passive_id":
			if value > 31 {
				return nil, fmt.Errorf("passive_id exceeds 31")
			}
			data[operation.Row].SetPassiveID(uint32(value))
			allowBytes(allowed, offset, 4)
		case "class_restriction":
			data[operation.Row].SetClassRestriction(uint32(value))
			allowBytes(allowed, offset+4, 4)
		case "effect_kind":
			data[operation.Row].SetEffectKind(uint32(value))
			allowBytes(allowed, offset+8, 4)
		case "level":
			if value > 6 {
				return nil, fmt.Errorf("level exceeds 6")
			}
			data[operation.Row].SetLevel(uint32(value))
			allowBytes(allowed, offset+12, 4)
		case "required_points":
			data[operation.Row].SetRequiredPoints(uint32(value))
			allowBytes(allowed, offset+16, 4)
		case "money":
			data[operation.Row].SetMoney(uint32(value))
			allowBytes(allowed, offset+20, 4)
		default:
			index, ok := fieldIndex(operation.Field, "effect_", passiveskillfile.EffectCount)
			if !ok {
				return nil, fmt.Errorf("unknown passive skill field %q", operation.Field)
			}
			if err := data[operation.Row].SetEffect(index, uint32(value)); err != nil {
				return nil, err
			}
			allowBytes(allowed, offset+24+index*4, 4)
		}
	}
	return writeZoneData(func(writer io.Writer) error { return passiveskillfile.Write(writer, data) })
}

func applyHiredSoldierSkill(original []byte, operations []ZoneDataOperation, allowed []bool) ([]byte, error) {
	data, err := hiredsoldierskillfile.Read(bytes.NewReader(original))
	if err != nil {
		return nil, err
	}
	for _, operation := range operations {
		if operation.Scope != "row" {
			return nil, fmt.Errorf("invalid hired-soldier skill scope %q", operation.Scope)
		}
		if err := validateRow(operation.Row, len(data)); err != nil {
			return nil, err
		}
		parts := strings.Split(operation.Field, "_")
		if len(parts) < 4 || parts[0] != "level" {
			return nil, fmt.Errorf("unknown hired-soldier skill field %q", operation.Field)
		}
		level, err := strconv.Atoi(parts[1])
		if err != nil || level < 1 || level > hiredsoldierskillfile.LevelCount {
			return nil, fmt.Errorf("invalid hired-soldier skill level")
		}
		field := strings.Join(parts[2:], "_")
		current, _ := data[operation.Row].Level(level)
		value, err := operationUint(operation, 4294967295)
		if err != nil {
			return nil, err
		}
		fieldOffset, fieldSize := 0, 0
		switch field {
		case "required_item_code":
			if value > 65535 {
				return nil, fmt.Errorf("required item code exceeds 65535")
			}
			current.RequiredItemCode = uint16(value)
			fieldOffset, fieldSize = 0x0a, 2
		case "skill_point_cost":
			if value > 255 {
				return nil, fmt.Errorf("skill point cost exceeds 255")
			}
			current.SkillPointCost = byte(value)
			fieldOffset, fieldSize = 0x0c, 1
		case "money_cost":
			current.MoneyCost = uint32(value)
			fieldOffset, fieldSize = 0x0d, 4
		case "lore_cost":
			current.LoreCost = uint32(value)
			fieldOffset, fieldSize = 0x11, 4
		default:
			return nil, fmt.Errorf("unknown hired-soldier skill field %q", operation.Field)
		}
		if err := data[operation.Row].SetLevel(level, current); err != nil {
			return nil, err
		}
		offset := operation.Row*hiredsoldierskillfile.RecordSize + (level-1)*hiredsoldierskillfile.LevelStride + fieldOffset
		allowBytes(allowed, offset, fieldSize)
	}
	return writeZoneData(func(writer io.Writer) error { return hiredsoldierskillfile.Write(writer, data) })
}

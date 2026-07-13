package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/project-agonyl/agonyl-utils-go/messagefile"
	"github.com/project-agonyl/agonyl-utils-go/oxquizfile"
	"github.com/project-agonyl/agonyl-utils-go/questexfile"
	"github.com/project-agonyl/agonyl-utils-go/squestquizfile"
	"github.com/project-agonyl/agonyl-utils-go/towertreasurefile"
	"github.com/project-agonyl/agonyl-utils-go/tyrbasefile"
	"github.com/project-agonyl/agonyl-utils-go/tyrgiftfile"
	"github.com/project-agonyl/agonyl-utils-go/tyrnpcregenfile"
	"github.com/project-agonyl/agonyl-utils-go/tyrportalfile"
	"github.com/project-agonyl/agonyl-utils-go/tyrskilllayerfile"
	"github.com/project-agonyl/agonyl-utils-go/tyrstartpointfile"
	"github.com/project-agonyl/agonyl-utils-go/tyrupgradefile"
)

func applyRemainingZoneData(original []byte, format ZoneDataFormat, operations []ZoneDataOperation, allowed []bool) ([]byte, error) {
	switch format {
	case ZoneDataFormatMessage:
		return applyMessages(original, operations, allowed)
	case ZoneDataFormatQuestEx:
		return applyQuestEx(original, operations, allowed)
	case ZoneDataFormatSQuestQuiz:
		return applySQuestQuiz(original, operations, allowed)
	case ZoneDataFormatTowerTreasure:
		return applyTowerTreasure(original, operations, allowed)
	case ZoneDataFormatOXQuiz:
		return applyOXQuiz(original, operations, allowed)
	case ZoneDataFormatTyrBase:
		return applyTyrUint16(original, operations, allowed, &tyrBaseCodec{})
	case ZoneDataFormatTyrPortal:
		return applyTyrUint16(original, operations, allowed, &tyrPortalCodec{})
	case ZoneDataFormatTyrUpgrade:
		return applyTyrUint16(original, operations, allowed, &tyrUpgradeCodec{})
	case ZoneDataFormatTyrStartPoint:
		return applyTyrUint16(original, operations, allowed, &tyrStartCodec{})
	case ZoneDataFormatTyrGift:
		return applyTyrGift(original, operations, allowed)
	case ZoneDataFormatTyrNPCRegen:
		return applyTyrNPCRegen(original, operations, allowed)
	case ZoneDataFormatTyrSkillLayer:
		return applyTyrSkillLayer(original, operations, allowed)
	default:
		return nil, fmt.Errorf("unsupported remaining ZoneData format %q", format)
	}
}

func applyMessages(original []byte, operations []ZoneDataOperation, allowed []bool) ([]byte, error) {
	data, err := messagefile.Read(bytes.NewReader(original))
	if err != nil {
		return nil, err
	}
	offsets := make([]int, len(data))
	offset := 2
	for i, r := range data {
		offsets[i] = offset
		offset += len(r.Raw)
	}
	for _, op := range operations {
		if err := validateRowOperation(op, len(data)); err != nil {
			return nil, err
		}
		switch op.Field {
		case "index":
			v, err := operationUint(op, messagefile.MaxRecords-1)
			if err != nil {
				return nil, err
			}
			data[op.Row].SetIndex(uint16(v))
			allowBytes(allowed, offsets[op.Row], 2)
		case "text":
			v, err := operationString(op)
			if err != nil {
				return nil, err
			}
			if len(v) != len(data[op.Row].Text()) {
				return nil, fmt.Errorf("message text length must remain %d bytes", len(data[op.Row].Text()))
			}
			if err := data[op.Row].SetText(v); err != nil {
				return nil, err
			}
			allowBytes(allowed, offsets[op.Row]+6, len(v))
		default:
			return nil, fmt.Errorf("unknown message field %q", op.Field)
		}
	}
	return writeZoneData(func(w io.Writer) error { return messagefile.Write(w, data) })
}

func applyQuestEx(original []byte, operations []ZoneDataOperation, allowed []bool) ([]byte, error) {
	data, err := questexfile.Read(bytes.NewReader(original))
	if err != nil {
		return nil, err
	}
	for _, op := range operations {
		if op.Scope != "row" || op.Row != 0 {
			return nil, fmt.Errorf("invalid QuestEx target")
		}
		found := false
		for _, field := range questExFields {
			if field.key != op.Field {
				continue
			}
			value, err := operationInt32(op)
			if err != nil {
				return nil, err
			}
			if err := data.SetInt32(field.offset, value); err != nil {
				return nil, err
			}
			physical := field.offset
			if field.offset >= questexfile.NextQuestOffset {
				physical = len(original) - (questexfile.NextQuestOffset + questexfile.NextQuestCount*4 - field.offset)
			}
			allowBytes(allowed, physical, 4)
			found = true
			break
		}
		if !found {
			return nil, fmt.Errorf("unknown QuestEx field %q", op.Field)
		}
	}
	return writeZoneData(func(w io.Writer) error { return questexfile.Write(w, data) })
}

func applySQuestQuiz(original []byte, operations []ZoneDataOperation, allowed []bool) ([]byte, error) {
	data, err := squestquizfile.Read(bytes.NewReader(original))
	if err != nil {
		return nil, err
	}
	for _, op := range operations {
		if err := validateRowOperation(op, len(data)); err != nil {
			return nil, err
		}
		offset := op.Row * squestquizfile.RecordSize
		switch op.Field {
		case "id":
			v, err := operationUint(op, 65535)
			if err != nil {
				return nil, err
			}
			data[op.Row].SetID(uint16(v))
			allowBytes(allowed, offset, 2)
		case "question":
			v, err := operationString(op)
			if err != nil {
				return nil, err
			}
			if err := data[op.Row].SetQuestion(v); err != nil {
				return nil, err
			}
			allowBytes(allowed, offset+2, squestquizfile.QuestionSize)
		case "correct":
			v, err := operationUint(op, squestquizfile.AnswerCount-1)
			if err != nil {
				return nil, err
			}
			if err := data[op.Row].SetCorrect(uint16(v)); err != nil {
				return nil, err
			}
			allowBytes(allowed, offset+0x260, 2)
		default:
			index, ok := fieldIndex(op.Field, "answer_", squestquizfile.AnswerCount)
			if !ok {
				return nil, fmt.Errorf("unknown SQuest quiz field %q", op.Field)
			}
			v, err := operationString(op)
			if err != nil {
				return nil, err
			}
			if err := data[op.Row].SetAnswer(index, v); err != nil {
				return nil, err
			}
			allowBytes(allowed, offset+0xcb+index*squestquizfile.AnswerSize, squestquizfile.AnswerSize)
		}
	}
	return writeZoneData(func(w io.Writer) error { return squestquizfile.Write(w, data) })
}

func applyTowerTreasure(original []byte, operations []ZoneDataOperation, allowed []bool) ([]byte, error) {
	data, err := towertreasurefile.Read(bytes.NewReader(original))
	if err != nil {
		return nil, err
	}
	for _, op := range operations {
		if err := validateRowOperation(op, len(data.Records)); err != nil {
			return nil, err
		}
		v, err := operationUint(op, 4294967295)
		if err != nil {
			return nil, err
		}
		offset := op.Row * towertreasurefile.RecordSize
		if op.Field == "item_code" {
			if v > 65535 {
				return nil, fmt.Errorf("item code exceeds 65535")
			}
			data.Records[op.Row].SetItemCode(uint16(v))
			allowBytes(allowed, offset, 2)
		} else if op.Field == "weight" {
			data.Records[op.Row].SetWeight(uint32(v))
			allowBytes(allowed, offset+2, 4)
		} else {
			return nil, fmt.Errorf("unknown Tower treasure field %q", op.Field)
		}
	}
	return writeZoneData(func(w io.Writer) error { return towertreasurefile.Write(w, data) })
}

func applyOXQuiz(original []byte, operations []ZoneDataOperation, allowed []bool) ([]byte, error) {
	data, err := oxquizfile.Read(bytes.NewReader(original))
	if err != nil {
		return nil, err
	}
	for _, op := range operations {
		if err := validateRowOperation(op, len(data.Records)); err != nil {
			return nil, err
		}
		offset := op.Row * oxquizfile.RecordSize
		switch op.Field {
		case "reward_code", "reward_count":
			v, err := operationUint(op, 65535)
			if err != nil {
				return nil, err
			}
			if op.Field == "reward_code" {
				data.Records[op.Row].SetRewardCode(uint16(v))
				allowBytes(allowed, offset, 2)
			} else {
				data.Records[op.Row].SetRewardCount(uint16(v))
				allowBytes(allowed, offset+2, 2)
			}
		case "answer":
			v, err := operationString(op)
			if err != nil || len(v) != 1 {
				return nil, fmt.Errorf("answer must be O or X")
			}
			if err := data.Records[op.Row].SetAnswer(v[0]); err != nil {
				return nil, err
			}
			allowBytes(allowed, offset+4, 1)
		case "question", "explanation":
			v, err := operationString(op)
			if err != nil {
				return nil, err
			}
			if op.Field == "question" {
				if err := data.Records[op.Row].SetQuestion(v); err != nil {
					return nil, err
				}
				allowBytes(allowed, offset+5, oxquizfile.TextSize)
			} else {
				if err := data.Records[op.Row].SetExplanation(v); err != nil {
					return nil, err
				}
				allowBytes(allowed, offset+0x45, oxquizfile.TextSize)
			}
		default:
			return nil, fmt.Errorf("unknown OXQuiz field %q", op.Field)
		}
	}
	return writeZoneData(func(w io.Writer) error { return oxquizfile.Write(w, data) })
}

type tyrUint16Codec interface {
	read([]byte) (int, int, error)
	set(int, int, uint16) error
	write(io.Writer) error
}

func applyTyrUint16(original []byte, operations []ZoneDataOperation, allowed []bool, codec tyrUint16Codec) ([]byte, error) {
	rows, size, err := codec.read(original)
	if err != nil {
		return nil, err
	}
	for _, op := range operations {
		if err := validateRowOperation(op, rows); err != nil {
			return nil, err
		}
		index, ok := fieldIndex(op.Field, "field_", size/2)
		if !ok {
			return nil, fmt.Errorf("unknown Tyr field %q", op.Field)
		}
		v, err := operationUint(op, 65535)
		if err != nil {
			return nil, err
		}
		if err := codec.set(op.Row, index, uint16(v)); err != nil {
			return nil, err
		}
		allowBytes(allowed, op.Row*size+index*2, 2)
	}
	return writeZoneData(codec.write)
}

type tyrBaseCodec struct{ data tyrbasefile.Data }

func (c *tyrBaseCodec) read(b []byte) (int, int, error) {
	d, e := tyrbasefile.Read(bytes.NewReader(b))
	c.data = d
	return len(d), tyrbasefile.RecordSize, e
}
func (c *tyrBaseCodec) set(r, f int, v uint16) error { return c.data[r].SetField(f, v) }
func (c *tyrBaseCodec) write(w io.Writer) error      { return tyrbasefile.Write(w, c.data) }

type tyrPortalCodec struct{ data tyrportalfile.Data }

func (c *tyrPortalCodec) read(b []byte) (int, int, error) {
	d, e := tyrportalfile.Read(bytes.NewReader(b))
	c.data = d
	return len(d), tyrportalfile.RecordSize, e
}
func (c *tyrPortalCodec) set(r, f int, v uint16) error { return c.data[r].SetField(f, v) }
func (c *tyrPortalCodec) write(w io.Writer) error      { return tyrportalfile.Write(w, c.data) }

type tyrUpgradeCodec struct{ data tyrupgradefile.Data }

func (c *tyrUpgradeCodec) read(b []byte) (int, int, error) {
	d, e := tyrupgradefile.Read(bytes.NewReader(b))
	c.data = d
	return len(d), tyrupgradefile.RecordSize, e
}
func (c *tyrUpgradeCodec) set(r, f int, v uint16) error { return c.data[r].SetField(f, v) }
func (c *tyrUpgradeCodec) write(w io.Writer) error      { return tyrupgradefile.Write(w, c.data) }

type tyrStartCodec struct{ data tyrstartpointfile.Data }

func (c *tyrStartCodec) read(b []byte) (int, int, error) {
	d, e := tyrstartpointfile.Read(bytes.NewReader(b))
	c.data = d
	return len(d), tyrstartpointfile.RecordSize, e
}
func (c *tyrStartCodec) set(r, f int, v uint16) error { return c.data[r].SetField(f, v) }
func (c *tyrStartCodec) write(w io.Writer) error      { return tyrstartpointfile.Write(w, c.data) }

func applyTyrGift(original []byte, operations []ZoneDataOperation, allowed []bool) ([]byte, error) {
	data, err := tyrgiftfile.Read(bytes.NewReader(original))
	if err != nil {
		return nil, err
	}
	for _, op := range operations {
		if err := validateRowOperation(op, len(data)); err != nil {
			return nil, err
		}
		v, err := operationUint(op, 4294967295)
		if err != nil {
			return nil, err
		}
		offset := op.Row * tyrgiftfile.RecordSize
		switch op.Field {
		case "item_code":
			data[op.Row].SetItemCode(uint32(v))
			allowBytes(allowed, offset, 4)
		case "count":
			data[op.Row].SetCount(uint32(v))
			allowBytes(allowed, offset+4, 4)
		case "weight":
			if v > tyrgiftfile.MaxWeight {
				return nil, fmt.Errorf("weight exceeds %d", tyrgiftfile.MaxWeight)
			}
			data[op.Row].SetWeight(uint32(v))
			allowBytes(allowed, offset+8, 4)
		default:
			return nil, fmt.Errorf("unknown Tyr gift field %q", op.Field)
		}
	}
	return writeZoneData(func(w io.Writer) error { return tyrgiftfile.Write(w, data) })
}

func applyTyrNPCRegen(original []byte, operations []ZoneDataOperation, allowed []bool) ([]byte, error) {
	data, err := tyrnpcregenfile.Read(bytes.NewReader(original))
	if err != nil {
		return nil, err
	}
	fields := map[string]int{"base_index": 0, "npc_type": 1, "x": 2, "y": 3, "direction": 4}
	for _, op := range operations {
		if err := validateRowOperation(op, len(data.Records)); err != nil {
			return nil, err
		}
		offset := op.Row * data.RecordSize
		if op.Field == "step" {
			if data.RecordSize != tyrnpcregenfile.FullRecordSize {
				return nil, fmt.Errorf("compact NPC regen has no step")
			}
			v, err := operationUint(op, 255)
			if err != nil {
				return nil, err
			}
			data.Records[op.Row].SetStep(byte(v))
			allowBytes(allowed, offset+10, 1)
			continue
		}
		index, ok := fields[op.Field]
		if !ok {
			return nil, fmt.Errorf("unknown Tyr NPC regen field %q", op.Field)
		}
		v, err := operationUint(op, 65535)
		if err != nil {
			return nil, err
		}
		if err := data.Records[op.Row].SetField(index, uint16(v)); err != nil {
			return nil, err
		}
		allowBytes(allowed, offset+index*2, 2)
	}
	return writeZoneData(func(w io.Writer) error { return tyrnpcregenfile.Write(w, data) })
}

func applyTyrSkillLayer(original []byte, operations []ZoneDataOperation, allowed []bool) ([]byte, error) {
	data, err := tyrskilllayerfile.Read(bytes.NewReader(original))
	if err != nil {
		return nil, err
	}
	for _, op := range operations {
		if err := validateRowOperation(op, len(data)); err != nil {
			return nil, err
		}
		offset := op.Row * tyrskilllayerfile.RecordSize
		switch op.Field {
		case "class_index", "skill_index":
			v, err := operationUint(op, 255)
			if err != nil {
				return nil, err
			}
			if op.Field == "class_index" {
				data[op.Row].SetClassIndex(uint16(v))
				allowBytes(allowed, offset, 2)
			} else {
				data[op.Row].SetSkillIndex(uint16(v))
				allowBytes(allowed, offset+2, 2)
			}
		default:
			var prefix string
			var base int
			if strings.HasPrefix(op.Field, "player_to_target_") {
				prefix = "player_to_target_"
				base = 4
			} else if strings.HasPrefix(op.Field, "target_to_effect_") {
				prefix = "target_to_effect_"
				base = 0x0e
			} else {
				return nil, fmt.Errorf("unknown Tyr skill layer field %q", op.Field)
			}
			index, err := strconv.Atoi(strings.TrimPrefix(op.Field, prefix))
			if err != nil || index < 0 || index >= tyrskilllayerfile.FlagCount {
				return nil, fmt.Errorf("unknown Tyr skill layer field %q", op.Field)
			}
			var v bool
			if err := json.Unmarshal(op.Value, &v); err != nil {
				return nil, fmt.Errorf("%s must be boolean", op.Field)
			}
			if prefix == "player_to_target_" {
				err = data[op.Row].SetPlayerToTarget(index, v)
			} else {
				err = data[op.Row].SetTargetToEffect(index, v)
			}
			if err != nil {
				return nil, err
			}
			allowBytes(allowed, offset+base+index*2, 2)
		}
	}
	return writeZoneData(func(w io.Writer) error { return tyrskilllayerfile.Write(w, data) })
}

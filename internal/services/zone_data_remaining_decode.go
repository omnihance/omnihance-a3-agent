package services

import (
	"bytes"
	"fmt"

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

var questExFields = []struct {
	key, label string
	offset     int
}{
	{"index", "Quest Index", 0}, {"start_npc", "Start NPC", 4}, {"end_npc", "End NPC", 8}, {"is_head", "Is Head", 12},
	{"required_class", "Required Class", 0x10}, {"required_item", "Required Item", 0x18}, {"previous_quest", "Previous Quest", 0x1c},
	{"minimum_level", "Minimum Level", 0x20}, {"maximum_level", "Maximum Level", 0x24}, {"required_favor", "Required Favor", 0x28},
	{"reward_experience", "Reward Experience", 0x50}, {"reward_money", "Reward Money", 0x54}, {"reward_lore", "Reward Lore", 0x58}, {"reward_favor", "Reward Favor", 0x5c},
	{"next_quest_0", "Next Quest 1", 0x664}, {"next_quest_1", "Next Quest 2", 0x668}, {"next_quest_2", "Next Quest 3", 0x66c},
}

func decodeRemainingZoneData(data []byte, format ZoneDataFormat) ([]ZoneDataField, []ZoneDataRow, error) {
	switch format {
	case ZoneDataFormatMessage:
		parsed, err := messagefile.Read(bytes.NewReader(data))
		if err != nil {
			return nil, nil, err
		}
		fields := []ZoneDataField{integerField("index", "Index", "row", messagefile.MaxRecords-1), stringField("text", "Text", "row")}
		rows := make([]ZoneDataRow, len(parsed))
		for i, r := range parsed {
			rows[i] = row(i, r.Raw, map[string]any{"index": r.Index(), "text": r.Text()})
		}
		return fields, rows, nil
	case ZoneDataFormatQuestEx:
		parsed, err := questexfile.Read(bytes.NewReader(data))
		if err != nil {
			return nil, nil, err
		}
		fields := make([]ZoneDataField, len(questExFields))
		values := make(map[string]any, len(fields))
		for i, f := range questExFields {
			fields[i] = signedField(f.key, f.label, "row")
			values[f.key] = parsed.Int32(f.offset)
		}
		return fields, []ZoneDataRow{row(0, parsed.Raw[:], values)}, nil
	case ZoneDataFormatSQuestQuiz:
		parsed, err := squestquizfile.Read(bytes.NewReader(data))
		if err != nil {
			return nil, nil, err
		}
		fields := []ZoneDataField{integerField("id", "ID", "row", 65535), stringField("question", "Question", "row")}
		for i := 0; i < squestquizfile.AnswerCount; i++ {
			fields = append(fields, stringField(fmt.Sprintf("answer_%d", i), fmt.Sprintf("Answer %d", i+1), "row"))
		}
		fields = append(fields, integerField("correct", "Correct Answer", "row", squestquizfile.AnswerCount-1))
		rows := make([]ZoneDataRow, len(parsed))
		for i, r := range parsed {
			v := map[string]any{"id": r.ID(), "question": r.Question(), "correct": r.Correct()}
			for a := 0; a < squestquizfile.AnswerCount; a++ {
				v[fmt.Sprintf("answer_%d", a)], _ = r.Answer(a)
			}
			rows[i] = row(i, r.Raw[:], v)
		}
		return fields, rows, nil
	case ZoneDataFormatTowerTreasure:
		parsed, err := towertreasurefile.Read(bytes.NewReader(data))
		if err != nil {
			return nil, nil, err
		}
		fields := []ZoneDataField{integerField("item_code", "Item Code", "row", 65535), integerField("weight", "Weight", "row", 4294967295)}
		rows := make([]ZoneDataRow, len(parsed.Records))
		for i, r := range parsed.Records {
			rows[i] = row(i, r.Raw[:], map[string]any{"item_code": r.ItemCode(), "weight": r.Weight()})
		}
		return fields, rows, nil
	case ZoneDataFormatOXQuiz:
		parsed, err := oxquizfile.Read(bytes.NewReader(data))
		if err != nil {
			return nil, nil, err
		}
		fields := []ZoneDataField{integerField("reward_code", "Reward Code", "row", 65535), integerField("reward_count", "Reward Count", "row", 65535), stringField("answer", "Answer", "row"), stringField("question", "Question", "row"), stringField("explanation", "Explanation", "row")}
		rows := make([]ZoneDataRow, len(parsed.Records))
		for i, r := range parsed.Records {
			rows[i] = row(i, r.Raw[:], map[string]any{"reward_code": r.RewardCode(), "reward_count": r.RewardCount(), "answer": string(r.Answer()), "question": r.Question(), "explanation": r.Explanation()})
		}
		return fields, rows, nil
	case ZoneDataFormatTyrBase:
		parsed, err := tyrbasefile.Read(bytes.NewReader(data))
		if err != nil {
			return nil, nil, err
		}
		raw, values := parsedToRawBase(parsed)
		return buildUint16Rows(raw, values, []string{"Index", "Grade", "War Point Value", "Morale Value", "Nation"})
	case ZoneDataFormatTyrPortal:
		parsed, err := tyrportalfile.Read(bytes.NewReader(data))
		if err != nil {
			return nil, nil, err
		}
		raw := make([][]byte, len(parsed))
		values := make([][]uint16, len(parsed))
		for i, r := range parsed {
			raw[i] = r.Raw[:]
			values[i] = []uint16{r.SourceX(), r.SourceY(), r.DestinationX(), r.DestinationY()}
		}
		return buildUint16Rows(raw, values, []string{"Source X", "Source Y", "Destination X", "Destination Y"})
	case ZoneDataFormatTyrUpgrade:
		parsed, err := tyrupgradefile.Read(bytes.NewReader(data))
		if err != nil {
			return nil, nil, err
		}
		raw := make([][]byte, len(parsed))
		values := make([][]uint16, len(parsed))
		labels := make([]string, tyrupgradefile.FieldCount)
		for i := range labels {
			labels[i] = fmt.Sprintf("Field %d", i)
		}
		for i, r := range parsed {
			raw[i] = r.Raw[:]
			values[i] = make([]uint16, tyrupgradefile.FieldCount)
			for f := range values[i] {
				values[i][f] = r.Field(f)
			}
		}
		return buildUint16Rows(raw, values, labels)
	case ZoneDataFormatTyrStartPoint:
		parsed, err := tyrstartpointfile.Read(bytes.NewReader(data))
		if err != nil {
			return nil, nil, err
		}
		raw := make([][]byte, len(parsed))
		values := make([][]uint16, len(parsed))
		for i, r := range parsed {
			raw[i] = r.Raw[:]
			values[i] = make([]uint16, tyrstartpointfile.FieldCount)
			for f := range values[i] {
				values[i][f] = r.Field(f)
			}
		}
		return buildUint16Rows(raw, values, []string{"Rank", "Unit", "Nation", "X", "Y", "Direction"})
	case ZoneDataFormatTyrGift:
		parsed, err := tyrgiftfile.Read(bytes.NewReader(data))
		if err != nil {
			return nil, nil, err
		}
		fields := []ZoneDataField{integerField("item_code", "Item Code", "row", 4294967295), integerField("count", "Count", "row", 4294967295), integerField("weight", "Weight", "row", tyrgiftfile.MaxWeight)}
		rows := make([]ZoneDataRow, len(parsed))
		for i, r := range parsed {
			rows[i] = row(i, r.Raw[:], map[string]any{"item_code": r.ItemCode(), "count": r.Count(), "weight": r.Weight()})
		}
		return fields, rows, nil
	case ZoneDataFormatTyrNPCRegen:
		parsed, err := tyrnpcregenfile.Read(bytes.NewReader(data))
		if err != nil {
			return nil, nil, err
		}
		fields := []ZoneDataField{integerField("base_index", "Base Index", "row", 15), integerField("npc_type", "NPC Type", "row", 65535), integerField("x", "X", "row", 65535), integerField("y", "Y", "row", 65535), integerField("direction", "Direction", "row", 8)}
		if parsed.RecordSize == tyrnpcregenfile.FullRecordSize {
			fields = append(fields, integerField("step", "Step", "row", 255))
		}
		rows := make([]ZoneDataRow, len(parsed.Records))
		for i, r := range parsed.Records {
			v := map[string]any{"base_index": r.BaseIndex(), "npc_type": r.NPCType(), "x": r.X(), "y": r.Y(), "direction": r.Direction()}
			if parsed.RecordSize == tyrnpcregenfile.FullRecordSize {
				v["step"] = r.Step()
			}
			rows[i] = row(i, r.Raw[:parsed.RecordSize], v)
		}
		return fields, rows, nil
	case ZoneDataFormatTyrSkillLayer:
		parsed, err := tyrskilllayerfile.Read(bytes.NewReader(data))
		if err != nil {
			return nil, nil, err
		}
		fields := []ZoneDataField{integerField("class_index", "Class", "row", 255), integerField("skill_index", "Skill", "row", 255)}
		for i := -2; i <= 2; i++ {
			fields = append(fields, boolField(fmt.Sprintf("player_to_target_%d", i+2), fmt.Sprintf("Player to Target Height %+d", i), "row"), boolField(fmt.Sprintf("target_to_effect_%d", i+2), fmt.Sprintf("Target to Effect Height %+d", i), "row"))
		}
		rows := make([]ZoneDataRow, len(parsed))
		for i, r := range parsed {
			v := map[string]any{"class_index": r.ClassIndex(), "skill_index": r.SkillIndex()}
			for f := 0; f < tyrskilllayerfile.FlagCount; f++ {
				v[fmt.Sprintf("player_to_target_%d", f)], _ = r.PlayerToTarget(f)
				v[fmt.Sprintf("target_to_effect_%d", f)], _ = r.TargetToEffect(f)
			}
			rows[i] = row(i, r.Raw[:], v)
		}
		return fields, rows, nil
	default:
		return nil, nil, fmt.Errorf("unsupported remaining ZoneData format %q", format)
	}
}

func signedField(key, label, scope string) ZoneDataField {
	min, max := int64(-2147483648), int64(2147483647)
	return ZoneDataField{Key: key, Label: label, Type: "integer", Scope: scope, Editable: true, Min: &min, Max: &max}
}
func parsedToRawBase(parsed tyrbasefile.Data) ([][]byte, [][]uint16) {
	raw := make([][]byte, len(parsed))
	values := make([][]uint16, len(parsed))
	for i, r := range parsed {
		raw[i] = r.Raw[:]
		values[i] = []uint16{r.Index(), r.Grade(), r.WarPointValue(), r.MoraleValue(), r.Nation()}
	}
	return raw, values
}
func buildUint16Rows(raw [][]byte, values [][]uint16, labels []string) ([]ZoneDataField, []ZoneDataRow, error) {
	fields := make([]ZoneDataField, len(labels))
	for i, label := range labels {
		fields[i] = integerField(fmt.Sprintf("field_%d", i), label, "row", 65535)
	}
	rows := make([]ZoneDataRow, len(raw))
	for i := range raw {
		v := make(map[string]any, len(labels))
		for f := range labels {
			v[fmt.Sprintf("field_%d", f)] = values[i][f]
		}
		rows[i] = row(i, raw[i], v)
	}
	return fields, rows, nil
}

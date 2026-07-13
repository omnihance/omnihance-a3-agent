package services

import (
	"bytes"
	"encoding/base64"
	"fmt"

	"github.com/project-agonyl/agonyl-utils-go/hiredsoldierskillfile"
	"github.com/project-agonyl/agonyl-utils-go/npcfavorfile"
	"github.com/project-agonyl/agonyl-utils-go/npcskillfile"
	"github.com/project-agonyl/agonyl-utils-go/passiveskillfile"
	"github.com/project-agonyl/agonyl-utils-go/pcdatafile"
	"github.com/project-agonyl/agonyl-utils-go/skilldatafile"
	"github.com/project-agonyl/agonyl-utils-go/skilldelayfile"
	"github.com/project-agonyl/agonyl-utils-go/zonemapfile"
)

var pcDataLabels = []string{
	"Strength", "Magic", "Dexterity", "Vitality", "Mana", "Bonus Points", "Attack", "Defense", "Magic Attack", "HP", "MP", "Hit Probability", "Damage Ratio", "Finish Ratio",
}

func mapSchema() []ZoneDataField {
	return []ZoneDataField{
		stringField("name", "Map Name", "map"),
		integerField("map_id", "Destination Map", "warp", 65535),
		integerField("cell", "Destination Cell", "warp", 65535),
		integerField("unknown", "Unknown", "warp", 65535),
		boolField("can_move", "Can Move", "cell"),
		integerField("pk_level", "PK Level", "cell", 3),
		integerField("warp_index", "Warp Index", "cell", 15),
	}
}

func npcSkillSchema() []ZoneDataField {
	return []ZoneDataField{
		integerField("npc_type", "NPC Type", "row", 65535), integerField("kind", "Kind", "row", 255),
		integerField("attack_type", "Attack Type", "row", 255), integerField("one_target_range", "One Target Range", "row", 255),
		integerField("range_radius", "Range Radius", "row", 255), integerField("cooldown_seconds", "Cooldown Seconds", "row", 65535),
		integerField("effect_param", "Effect Parameter", "row", 65535), integerField("effect_code", "Effect Code", "row", 65535),
		integerField("effect_value", "Effect Value", "row", 65535),
	}
}

func npcFavorSchema() []ZoneDataField {
	min := int64(-2147483648)
	max := int64(2147483647)
	index := ZoneDataField{Key: "index", Label: "Favor Index", Type: "integer", Scope: "row", Editable: true, Min: &min, Max: &max}
	return []ZoneDataField{index, integerField("npc_type", "NPC Type", "row", 4294967295)}
}

func pcDataSchema() []ZoneDataField {
	fields := make([]ZoneDataField, len(pcDataLabels))
	for index, label := range pcDataLabels {
		fields[index] = integerField(fmt.Sprintf("value_%d", index), label, "row", 65535)
	}

	return fields
}

func skillDataSchema() []ZoneDataField {
	fields := []ZoneDataField{
		integerField("code", "Code", "row", 255), integerField("type", "Type", "row", 255), integerField("sub_type", "Sub Type", "row", 255),
		integerField("target_type", "Target Type", "row", 255), integerField("need_item", "Required Item", "row", 255),
		integerField("reaction", "Reaction", "row", 255), integerField("abnormalcy", "Abnormalcy", "row", 255),
	}
	for index := 0; index < 6; index++ {
		fields = append(fields, integerField(fmt.Sprintf("monster_rate_%d", index), fmt.Sprintf("Monster Rate %d", index+1), "row", 255))
	}

	return fields
}

func skillDelaySchema() []ZoneDataField {
	return []ZoneDataField{
		integerField("class_index", "Class", "row", 3), integerField("skill_index", "Skill", "row", 63),
		integerField("delay_ms", "Delay", "row", 65535), integerField("delay_info", "Delay Info", "row", 255),
	}
}

func passiveSkillSchema() []ZoneDataField {
	fields := []ZoneDataField{
		integerField("passive_id", "Passive ID", "row", 31), integerField("class_restriction", "Class Restriction", "row", 255),
		integerField("effect_kind", "Effect Kind", "row", 255), integerField("level", "Level", "row", 6),
		integerField("required_points", "Required Points", "row", 255), integerField("money", "Money", "row", 4294967295),
	}
	for index := 0; index < passiveskillfile.EffectCount; index++ {
		fields = append(fields, integerField(fmt.Sprintf("effect_%d", index), fmt.Sprintf("Effect %d", index+1), "row", 4294967295))
	}

	return fields
}

func hiredSoldierSkillSchema() []ZoneDataField {
	fields := make([]ZoneDataField, 0, hiredsoldierskillfile.LevelCount*4)
	for level := 1; level <= hiredsoldierskillfile.LevelCount; level++ {
		prefix := fmt.Sprintf("level_%d_", level)
		fields = append(fields,
			integerField(prefix+"required_item_code", fmt.Sprintf("Level %d Item", level), "row", 65535),
			integerField(prefix+"skill_point_cost", fmt.Sprintf("Level %d Skill Points", level), "row", 255),
			integerField(prefix+"money_cost", fmt.Sprintf("Level %d Money", level), "row", 4294967295),
			integerField(prefix+"lore_cost", fmt.Sprintf("Level %d Lore", level), "row", 4294967295),
		)
	}

	return fields
}

func decodeMap(data []byte) (*ZoneDataMap, error) {
	parsed, err := zonemapfile.Read(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	warps := make([]ZoneDataRow, len(parsed.Warps))
	for index, warp := range parsed.Warps {
		warps[index] = row(index, warp.Raw[:], map[string]any{"map_id": warp.MapID(), "cell": warp.Cell(), "unknown": warp.Unknown()})
	}

	cells := make([]uint32, len(parsed.Cells))
	copy(cells, parsed.Cells[:])
	return &ZoneDataMap{
		Name: parsed.Name(), Warps: warps, Cells: cells, Width: zonemapfile.Width, Height: zonemapfile.Height,
		Trailing: base64Bytes(parsed.Trailing),
	}, nil
}

func decodeNPCSkill(data []byte) ([]ZoneDataRow, error) {
	parsed, err := npcskillfile.Read(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	rows := make([]ZoneDataRow, len(parsed))
	for index, record := range parsed {
		rows[index] = row(index, record.Raw[:], map[string]any{
			"npc_type": record.NPCType(), "kind": record.Kind(), "attack_type": record.AttackType(), "one_target_range": record.OneTargetRange(),
			"range_radius": record.RangeRadius(), "cooldown_seconds": record.CooldownSeconds(), "effect_param": record.EffectParam(),
			"effect_code": record.EffectCode(), "effect_value": record.EffectValue(),
		})
	}

	return rows, nil
}

func decodeNPCFavor(data []byte) ([]ZoneDataRow, error) {
	parsed, err := npcfavorfile.Read(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	rows := make([]ZoneDataRow, len(parsed))
	for index, record := range parsed {
		rows[index] = row(index, record.Raw[:], map[string]any{"index": record.Index(), "npc_type": record.NPCType()})
	}

	return rows, nil
}

func decodePCData(data []byte) ([]ZoneDataRow, error) {
	parsed, err := pcdatafile.Read(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	values := make(map[string]any, len(pcDataLabels))
	for index := range pcDataLabels {
		values[fmt.Sprintf("value_%d", index)], _ = parsed.Value(index)
	}

	return []ZoneDataRow{row(0, parsed.Raw[:], values)}, nil
}

func decodeSkillData(data []byte) ([]ZoneDataRow, error) {
	parsed, err := skilldatafile.Read(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	rows := make([]ZoneDataRow, len(parsed))
	for index, record := range parsed {
		values := map[string]any{
			"code": record.Code(), "type": record.Type(), "sub_type": record.SubType(), "target_type": record.TargetType(),
			"need_item": record.NeedItem(), "reaction": record.Reaction(), "abnormalcy": record.Abnormalcy(),
		}
		for rateIndex := 0; rateIndex < 6; rateIndex++ {
			values[fmt.Sprintf("monster_rate_%d", rateIndex)], _ = record.MonsterRate(rateIndex)
		}
		rows[index] = row(index, record.Raw[:], values)
	}

	return rows, nil
}

func decodeSkillDelay(data []byte) ([]ZoneDataRow, error) {
	parsed, err := skilldelayfile.Read(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	rows := make([]ZoneDataRow, len(parsed))
	for index, record := range parsed {
		rows[index] = row(index, record.Raw[:], map[string]any{
			"class_index": record.ClassIndex(), "skill_index": record.SkillIndex(), "delay_ms": record.DelayMS(), "delay_info": record.DelayInfo(),
		})
	}

	return rows, nil
}

func decodePassiveSkill(data []byte) ([]ZoneDataRow, error) {
	parsed, err := passiveskillfile.Read(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	rows := make([]ZoneDataRow, len(parsed))
	for index, record := range parsed {
		values := map[string]any{
			"passive_id": record.PassiveID(), "class_restriction": record.ClassRestriction(), "effect_kind": record.EffectKind(),
			"level": record.Level(), "required_points": record.RequiredPoints(), "money": record.Money(),
		}
		for effectIndex := 0; effectIndex < passiveskillfile.EffectCount; effectIndex++ {
			values[fmt.Sprintf("effect_%d", effectIndex)], _ = record.Effect(effectIndex)
		}
		rows[index] = row(index, record.Raw[:], values)
	}

	return rows, nil
}

func decodeHiredSoldierSkill(data []byte) ([]ZoneDataRow, error) {
	parsed, err := hiredsoldierskillfile.Read(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	rows := make([]ZoneDataRow, len(parsed))
	for index, record := range parsed {
		values := make(map[string]any, hiredsoldierskillfile.LevelCount*4)
		for level := 1; level <= hiredsoldierskillfile.LevelCount; level++ {
			value, _ := record.Level(level)
			prefix := fmt.Sprintf("level_%d_", level)
			values[prefix+"required_item_code"] = value.RequiredItemCode
			values[prefix+"skill_point_cost"] = value.SkillPointCost
			values[prefix+"money_cost"] = value.MoneyCost
			values[prefix+"lore_cost"] = value.LoreCost
		}
		rows[index] = row(index, record.Raw[:], values)
	}

	return rows, nil
}

func base64Bytes(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	return base64.StdEncoding.EncodeToString(data)
}

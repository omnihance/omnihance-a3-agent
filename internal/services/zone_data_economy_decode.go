package services

import (
	"bytes"
	"fmt"

	"github.com/project-agonyl/agonyl-utils-go/a3presentfile"
	"github.com/project-agonyl/agonyl-utils-go/cashitemfile"
	"github.com/project-agonyl/agonyl-utils-go/derbygiftfile"
	"github.com/project-agonyl/agonyl-utils-go/eventitemrewardfile"
	"github.com/project-agonyl/agonyl-utils-go/lotteryfile"
	"github.com/project-agonyl/agonyl-utils-go/petfile"
	"github.com/project-agonyl/agonyl-utils-go/presentitemsetfile"
	"github.com/project-agonyl/agonyl-utils-go/setitemfile"
	"github.com/project-agonyl/agonyl-utils-go/shuecombinationfile"
)

var shueCombinationLabels = []string{
	"Base Rune 1", "Base Rune 2", "Base Rune 3", "Rune 1", "Rune 2", "Rune 3", "Item 1", "Item 2", "Item 3", "Object Item", "Success Ratio", "Success Item", "Option Type", "Failure", "Reserved 1", "Reserved 2",
}

func decodeEconomyZoneData(data []byte, format ZoneDataFormat) ([]ZoneDataField, []ZoneDataRow, error) {
	switch format {
	case ZoneDataFormatCashItem:
		return decodeCashItems(data)
	case ZoneDataFormatSetItem:
		return decodeSetItems(data)
	case ZoneDataFormatPresentItemSet:
		return decodePresentItemSets(data)
	case ZoneDataFormatPet:
		return decodePets(data)
	case ZoneDataFormatShueCombination:
		return decodeShueCombinations(data)
	case ZoneDataFormatLottery:
		return decodeLottery(data)
	case ZoneDataFormatDerbyGift:
		return decodeDerbyGifts(data)
	case ZoneDataFormatEventItemReward:
		return decodeEventItemRewards(data)
	case ZoneDataFormatA3Present:
		return decodeA3Presents(data)
	default:
		return nil, nil, fmt.Errorf("unsupported economy ZoneData format %q", format)
	}
}

func decodeCashItems(data []byte) ([]ZoneDataField, []ZoneDataRow, error) {
	parsed, err := cashitemfile.Read(bytes.NewReader(data))
	if err != nil {
		return nil, nil, err
	}

	fields := []ZoneDataField{
		integerField("npc_type", "NPC Type", "row", 65535), integerField("item_code", "Item Code", "row", 65535),
		integerField("price", "Price", "row", 4294967295), integerField("count", "Count", "row", 65535),
	}
	rows := make([]ZoneDataRow, len(parsed))
	for index, record := range parsed {
		rows[index] = row(index, record.Raw[:], map[string]any{"npc_type": record.NPCType(), "item_code": record.ItemCode(), "price": record.Price(), "count": record.Count()})
	}

	return fields, rows, nil
}

func decodeSetItems(data []byte) ([]ZoneDataField, []ZoneDataRow, error) {
	parsed, err := setitemfile.Read(bytes.NewReader(data))
	if err != nil {
		return nil, nil, err
	}

	fields := make([]ZoneDataField, 0, setitemfile.PieceCount*2+(setitemfile.BonusPieceCount-1)*setitemfile.BonusCount*2)
	for index := 0; index < setitemfile.PieceCount; index++ {
		fields = append(fields,
			integerField(fmt.Sprintf("piece_%d_wear_slot", index), fmt.Sprintf("Piece %d Wear Slot", index+1), "row", 255),
			integerField(fmt.Sprintf("piece_%d_code", index), fmt.Sprintf("Piece %d Code", index+1), "row", 65535),
		)
	}
	for pieceCount := 1; pieceCount < setitemfile.BonusPieceCount; pieceCount++ {
		for index := 0; index < setitemfile.BonusCount; index++ {
			fields = append(fields,
				integerField(fmt.Sprintf("bonus_%d_%d_option_id", pieceCount, index), fmt.Sprintf("%d Pieces Bonus %d Option", pieceCount, index+1), "row", 255),
				integerField(fmt.Sprintf("bonus_%d_%d_value", pieceCount, index), fmt.Sprintf("%d Pieces Bonus %d Value", pieceCount, index+1), "row", 65535),
			)
		}
	}
	rows := make([]ZoneDataRow, len(parsed))
	for rowIndex, record := range parsed {
		values := make(map[string]any, len(fields))
		for index := 0; index < setitemfile.PieceCount; index++ {
			wearSlot, code, _ := record.Piece(index)
			values[fmt.Sprintf("piece_%d_wear_slot", index)] = wearSlot
			values[fmt.Sprintf("piece_%d_code", index)] = code
		}
		for pieceCount := 1; pieceCount < setitemfile.BonusPieceCount; pieceCount++ {
			for index := 0; index < setitemfile.BonusCount; index++ {
				optionID, value, _ := record.Bonus(pieceCount, index)
				values[fmt.Sprintf("bonus_%d_%d_option_id", pieceCount, index)] = optionID
				values[fmt.Sprintf("bonus_%d_%d_value", pieceCount, index)] = value
			}
		}
		rows[rowIndex] = row(rowIndex, record.Raw[:], values)
	}

	return fields, rows, nil
}

func decodePresentItemSets(data []byte) ([]ZoneDataField, []ZoneDataRow, error) {
	parsed, err := presentitemsetfile.Read(bytes.NewReader(data))
	if err != nil {
		return nil, nil, err
	}

	fields := []ZoneDataField{integerField("present_set_code", "Present Set Code", "row", 65535)}
	for index := 0; index < presentitemsetfile.RewardCount; index++ {
		fields = append(fields, integerField(fmt.Sprintf("reward_%d_count", index), fmt.Sprintf("Reward %d Count", index+1), "row", 65535), integerField(fmt.Sprintf("reward_%d_item_code", index), fmt.Sprintf("Reward %d Item", index+1), "row", 65535))
	}
	rows := make([]ZoneDataRow, len(parsed))
	for rowIndex, record := range parsed {
		values := map[string]any{"present_set_code": record.PresentSetCode()}
		for index := 0; index < presentitemsetfile.RewardCount; index++ {
			count, itemCode, _ := record.Reward(index)
			values[fmt.Sprintf("reward_%d_count", index)] = count
			values[fmt.Sprintf("reward_%d_item_code", index)] = itemCode
		}
		rows[rowIndex] = row(rowIndex, record.Raw[:], values)
	}

	return fields, rows, nil
}

func decodePets(data []byte) ([]ZoneDataField, []ZoneDataRow, error) {
	parsed, err := petfile.Read(bytes.NewReader(data))
	if err != nil {
		return nil, nil, err
	}

	fields := []ZoneDataField{
		integerField("group", "Group", "row", 65535), integerField("code", "Code", "row", petfile.CodeMax), stringField("name", "Name", "row"),
		integerField("price", "Price", "row", 4294967295), integerField("auth", "Authorization", "row", 65535), integerField("limit", "Limit", "row", 65535),
		integerField("data_1", "Data 1", "row", 65535), integerField("data_2", "Data 2", "row", 65535),
	}
	rows := make([]ZoneDataRow, len(parsed))
	for index, record := range parsed {
		rows[index] = row(index, record.Raw[:], map[string]any{"group": record.Group(), "code": record.Code(), "name": record.Name(), "price": record.Price(), "auth": record.Auth(), "limit": record.Limit(), "data_1": record.Data1(), "data_2": record.Data2()})
	}

	return fields, rows, nil
}

func decodeShueCombinations(data []byte) ([]ZoneDataField, []ZoneDataRow, error) {
	parsed, err := shuecombinationfile.Read(bytes.NewReader(data))
	if err != nil {
		return nil, nil, err
	}

	fields := make([]ZoneDataField, shuecombinationfile.FieldCount)
	for index, label := range shueCombinationLabels {
		fields[index] = integerField(fmt.Sprintf("field_%d", index), label, "row", 65535)
	}
	rows := make([]ZoneDataRow, len(parsed))
	for rowIndex, record := range parsed {
		values := make(map[string]any, shuecombinationfile.FieldCount)
		for index := 0; index < shuecombinationfile.FieldCount; index++ {
			values[fmt.Sprintf("field_%d", index)], _ = record.Field(index)
		}
		rows[rowIndex] = row(rowIndex, record.Raw[:], values)
	}

	return fields, rows, nil
}

func decodeLottery(data []byte) ([]ZoneDataField, []ZoneDataRow, error) {
	parsed, err := lotteryfile.Read(bytes.NewReader(data))
	if err != nil {
		return nil, nil, err
	}

	fields := []ZoneDataField{integerField("event_item_id", "Event Item ID", "row", 255), integerField("enabled", "Enabled", "row", 255), integerField("reward_item_code", "Reward Item", "row", 65535)}
	for index := 0; index < lotteryfile.MessageCount; index++ {
		fields = append(fields, stringField(fmt.Sprintf("message_%d", index), fmt.Sprintf("Message %d", index+1), "row"))
	}
	rows := make([]ZoneDataRow, len(parsed))
	for rowIndex, record := range parsed {
		values := map[string]any{"event_item_id": record.EventItemID(), "enabled": record.Enabled(), "reward_item_code": record.RewardItemCode()}
		for index := 0; index < lotteryfile.MessageCount; index++ {
			values[fmt.Sprintf("message_%d", index)], _ = record.Message(index)
		}
		rows[rowIndex] = row(rowIndex, record.Raw[:], values)
	}

	return fields, rows, nil
}

func decodeDerbyGifts(data []byte) ([]ZoneDataField, []ZoneDataRow, error) {
	parsed, err := derbygiftfile.Read(bytes.NewReader(data))
	if err != nil {
		return nil, nil, err
	}

	fields := []ZoneDataField{integerField("item_code", "Item Code", "row", 4294967295), integerField("quantity", "Quantity", "row", 4294967295), integerField("weight", "Probability Weight", "row", derbygiftfile.MaxProbability)}
	rows := make([]ZoneDataRow, len(parsed))
	for index, record := range parsed {
		rows[index] = row(index, record.Raw[:], map[string]any{"item_code": record.ItemCode(), "quantity": record.Quantity(), "weight": record.Weight()})
	}

	return fields, rows, nil
}

func decodeEventItemRewards(data []byte) ([]ZoneDataField, []ZoneDataRow, error) {
	parsed, err := eventitemrewardfile.Read(bytes.NewReader(data))
	if err != nil {
		return nil, nil, err
	}

	fields := []ZoneDataField{integerField("item_code", "Item Code", "row", 65535), integerField("weight", "Probability Weight", "row", 65535), stringField("message", "Message", "row")}
	rows := make([]ZoneDataRow, len(parsed))
	for index, record := range parsed {
		rows[index] = row(index, record.Raw[:], map[string]any{"item_code": record.ItemCode(), "weight": record.Weight(), "message": record.Message()})
	}

	return fields, rows, nil
}

func decodeA3Presents(data []byte) ([]ZoneDataField, []ZoneDataRow, error) {
	parsed, err := a3presentfile.Read(bytes.NewReader(data))
	if err != nil {
		return nil, nil, err
	}

	fields := []ZoneDataField{stringField("name", "Character Name", "row")}
	for index := 0; index < a3presentfile.RewardCount; index++ {
		fields = append(fields, integerField(fmt.Sprintf("reward_%d_count", index), fmt.Sprintf("Reward %d Count", index+1), "row", 65535), integerField(fmt.Sprintf("reward_%d_item_code", index), fmt.Sprintf("Reward %d Item", index+1), "row", 65535))
	}
	fields = append(fields, integerField("money", "Money", "row", 4294967295), integerField("lore", "Lore", "row", 4294967295), integerField("experience", "Experience", "row", 4294967295), integerField("offered", "Offered", "row", 65535))
	rows := make([]ZoneDataRow, len(parsed))
	for rowIndex, record := range parsed {
		values := map[string]any{"name": record.Name(), "money": record.Money(), "lore": record.Lore(), "experience": record.Experience(), "offered": record.Offered()}
		for index := 0; index < a3presentfile.RewardCount; index++ {
			count, itemCode, _ := record.Reward(index)
			values[fmt.Sprintf("reward_%d_count", index)] = count
			values[fmt.Sprintf("reward_%d_item_code", index)] = itemCode
		}
		rows[rowIndex] = row(rowIndex, record.Raw[:], values)
	}

	return fields, rows, nil
}

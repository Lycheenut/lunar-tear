package missionprogress

// Quest-clear option IDs are backend condition-group IDs. They are not quest
// IDs, even though many of their numeric values happen to collide with one.
var mainQuestTargetsByOption = map[int32][]int32{
	6: {32}, 10: {1}, 11: {13}, 12: {31},
	13: {315}, 14: {318}, 15: {321}, 16: {324},
	17: {415}, 18: {418}, 19: {421}, 20: {424},
	25: {10025},
	32: {325}, 33: {328}, 34: {331}, 35: {334},
	36: {425}, 37: {428}, 38: {431}, 39: {434},
	44: {341}, 45: {344}, 46: {347}, 47: {350},
	48: {441}, 49: {444}, 50: {447}, 51: {450},
	56: {351}, 57: {354}, 58: {357}, 59: {359}, 60: {360},
	61: {451}, 62: {454}, 63: {457}, 64: {459}, 65: {460},
	72: {361}, 73: {364}, 74: {367}, 75: {369},
	76: {461}, 77: {464}, 78: {467}, 79: {469},

	101: {11}, 102: {21}, 103: {31}, 104: {41},
	105: {51}, 106: {61}, 107: {71}, 108: {81},
	109: {91}, 110: {101}, 111: {111}, 112: {121},
	113: {314}, 114: {414}, 115: {324}, 116: {424},
	117: {334}, 118: {434}, 119: {350}, 120: {450},
	121: {360}, 122: {460}, 123: {370}, 124: {470},
	125: {507}, 126: {514}, 127: {521}, 128: {529},
	129: {536}, 130: {546}, 131: {550},

	10001: {10004}, 10002: {10014}, 10003: {10024}, 10004: {10034},
	10005: {10044}, 10006: {10054}, 10007: {10064}, 10008: {10074},
	10009: {10084}, 10010: {10094}, 10011: {10104}, 10012: {10108},
	10013: {10310}, 10014: {10410}, 10015: {10320}, 10016: {10420},
	10017: {10330}, 10018: {10430}, 10019: {10340}, 10020: {10440},
	10021: {10350}, 10022: {10450}, 10023: {10360}, 10024: {10460},
	10025: {10507}, 10026: {10514}, 10027: {10521}, 10028: {10529},
	10029: {10536}, 10030: {10546},

	20001: {20004}, 20002: {20014}, 20003: {20024}, 20004: {20034},
	20005: {20044}, 20006: {20054}, 20007: {20064}, 20008: {20074},
	20009: {20084}, 20010: {20094}, 20011: {20104}, 20012: {20108},
	20013: {20310}, 20014: {20410}, 20015: {20320}, 20016: {20420},
	20017: {20330}, 20018: {20430}, 20019: {20340}, 20020: {20440},
	20021: {20350}, 20022: {20450}, 20023: {20360}, 20024: {20460},
	20025: {20507}, 20026: {20514}, 20027: {20521}, 20028: {20529},
	20029: {20536}, 20030: {20546},

	// These two IDs collide with old Event Quest IDs.
	100022: {10035},
	100023: {10050},
}

var mainQuestTargetsByDetail = map[int32][]int32{
	460002: {41},
	500056: {20034},
	500057: {120040},
	500058: {10064},
	500059: {10074},
	500084: {20044},
	500085: {120050},
	500087: {10094},
}

var specificEventQuestTargetsByOption = map[int32][]int32{
	// Dungeon: The Dynast's Memories 10F. The option is a condition-group
	// ID, while the actual Event Quest ID is 210010.
	500004: {210010},
}

// These early and once-per-day missions store a real QuestId directly in the
// option field. Keep the audited list explicit so an unrelated condition-group
// ID cannot start matching merely because a future quest reuses the number.
var directQuestOptions = idSet(
	30001, 30005, 30009, 30013, 30017, 30021,
	30031, 30032, 30033, 30034, 30035, 30036, 30037, 30038,
	40001, 40002, 40005, 40006, 40009, 40010,
	40013, 40014, 40017, 40018, 40021, 40022,
	50001, 50002, 50003, 50004, 50005, 50011,
)

var requiredDeckCharacterByOption = map[int32]int32{
	479: 1008, 480: 1010, 481: 1011,
	501: 1009, 502: 1012,
	500001: 1013, 500002: 1008, 500003: 1022, 500004: 1022,
	500015: 1009, 500026: 1012, 500032: 1023, 500038: 1015,
	500077: 1006, 500095: 1004, 500099: 1019, 500113: 1024,
	500124: 1007, 500146: 1010, 50004501: 1014,
}

var requiredDeckCostumeByOption = map[int32]int32{
	377: 35010, 390: 35012, 408: 22006, 430: 31013,
	434: 24003, 435: 25005, 436: 24004,
	449: 24007, 450: 21003, 459: 22007,
	460: 23004, 461: 21001, 462: 22004, 463: 32001,
	468: 23005, 482: 21004, 491: 22008, 496: 22005,
	503: 25006, 508: 24006, 517: 25007, 518: 35008,
	531: 34029, 542: 32018, 544: 24005, 557: 25008, 570: 22009,
	900001: 25002, 900003: 24006, 900005: 25003, 900006: 24005,
}

var requiredSoloCharacterByDetail = map[int32]int32{
	500056: 1015,
	500057: 1015,
	500058: 1011,
	500059: 1012,
	500084: 1006,
	500085: 1006,
	500087: 1014,
}

var requiredBigHuntCharactersByDetail = map[int32][]int32{
	500045: {1013, 1014},
	500067: {1011, 1012},
	500068: {1011, 1012},
	500078: {1022},
	500089: {1013, 1014},
	500102: {1004},
	500108: {1020},
	500109: {1021},
}

const (
	eventQuestDifficultyNormal   = int32(1)
	eventQuestDifficultyHard     = int32(2)
	eventQuestDifficultyVeryHard = int32(3)
	eventQuestDifficultyExHard   = int32(4)
)

type eventQuestSelector struct {
	difficulty int32
	ordinal    int32
	sortOrder  int32
	last       bool
	all        bool
}

func idSet(ids ...int32) map[int32]bool {
	result := make(map[int32]bool, len(ids))
	for _, id := range ids {
		result[id] = true
	}
	return result
}

var eventNormalLastOptions = idSet(
	193, 213, 233, 250, 260, 277, 287, 311, 331, 341, 355, 369, 382, 400,
	422, 431, 441, 451, 473, 483, 497, 509, 522, 536, 545, 549, 562, 575,
	588, 594, 596, 602, 605, 612, 618, 624, 631, 643, 646, 653, 656, 659,
	666, 672, 675, 678, 684, 687, 693, 696, 705, 708, 711, 714,
	200001, 200007, 200010, 200013,
)

var eventHardLastOptions = idSet(
	194, 214, 234, 251, 261, 278, 288, 312, 332, 342, 356, 370, 383, 401,
	423, 432, 442, 452, 474, 484, 498, 510, 523, 537, 546, 550, 563,
	200002, 200008, 200011, 200014,
)

var eventVeryHardLastOptions = idSet(
	195, 215, 235, 252, 262, 279, 289, 313, 333, 343, 357, 371, 384, 402,
	424, 433, 443, 453, 475, 485, 499, 511, 524, 538, 547, 551, 564,
	200003, 200009, 200012, 200015,
)

var eventAllQuestOptions = idSet(
	196, 204, 216, 224, 236, 243, 253, 263, 270, 280, 290, 297, 304, 314,
	334, 344, 351, 358, 365, 372, 378, 385, 396, 403, 409, 425, 437, 444,
	454, 464, 469, 476, 486, 492, 500, 504, 512, 525, 532, 539, 548, 552,
	558, 565, 571, 577, 586, 590, 593, 598, 604, 607, 610, 614, 620, 623,
	626, 629, 633, 637, 642, 645, 648, 652, 655, 658, 661, 664, 668, 671,
	674, 677, 680, 683, 686, 689, 692, 695, 698, 701, 707, 710, 713, 716,
	200017,
)

var eventNormalFirstOptions = idSet(
	584, 591, 608, 616, 621, 627, 635, 638, 640, 650, 662, 669, 681, 690, 699,
)

var eventExHardFifthOptions = idSet(
	576, 585, 587, 589, 592, 595, 597, 599, 603, 606, 609, 611, 613, 615,
	617, 619, 622, 625, 628, 630, 632, 636, 639, 641, 644, 647, 649, 651,
	654, 657, 660, 663, 667, 670, 673, 676, 679, 682, 685, 688, 691, 694,
	700, 706, 709, 712, 715, 200016,
)

var genericEventOptionsWithoutChapter = idSet(196, 216, 236, 11009, 11010, 11011)

var fallbackEventChapterIdsByOption = map[int32][]int32{
	21: {99015}, 22: {99015}, 23: {99015}, 24: {99015},
	40: {99016}, 41: {99016}, 42: {99016}, 43: {99016},
	52: {99017}, 53: {99017}, 54: {99017}, 55: {99017},
	66: {99018}, 67: {99018}, 68: {99018}, 69: {99018},
	80: {99019}, 81: {99019}, 82: {99019}, 83: {99019},
	11001: {912}, 11002: {912}, 11003: {912},
	11004: {906}, 11005: {906}, 11006: {906},
	101120401: {905}, 101120501: {905},
	431: {510}, 432: {510}, 433: {510},
	594: {531}, 595: {531},
	611: {544}, 615: {318}, 630: {526},
	200016: {510}, 200017: {510}, 200018: {502},
}

var characterQuestSortOrderByOption = map[int32]int32{
	21: 1, 22: 4, 23: 7, 24: 10,
	40: 1, 41: 4, 42: 7, 43: 10,
	52: 1, 53: 4, 54: 7, 55: 10,
	66: 1, 67: 4, 68: 7, 69: 10,
	80: 1, 81: 4, 82: 7, 83: 10,
}

var darkMemorySortOrderByOption = map[int32]int32{
	11001: 8, 11002: 9, 11003: 10,
	11004: 8, 11005: 9, 11006: 10,
	11009: 8, 11010: 9, 11011: 10,
	101120401: 1,
	101120501: 4,
}

var limitContentCharacterByOption = map[int32]int32{
	11007: 1013,
	11008: 1019,
	11012: 1011,
	11013: 1015,
	11014: 1014,
	11015: 1012,
	11016: 1006,
}

var darkMemoryCharacterByOption = map[int32]int32{
	500144: 1024, // Marie
	500159: 1020, // Hina
}

func eventSelectorForOption(option int32) (eventQuestSelector, bool) {
	switch {
	case characterQuestSortOrderByOption[option] != 0:
		return eventQuestSelector{sortOrder: characterQuestSortOrderByOption[option]}, true
	case darkMemorySortOrderByOption[option] != 0:
		return eventQuestSelector{sortOrder: darkMemorySortOrderByOption[option]}, true
	case eventNormalLastOptions[option]:
		return eventQuestSelector{difficulty: eventQuestDifficultyNormal, last: true}, true
	case eventHardLastOptions[option]:
		return eventQuestSelector{difficulty: eventQuestDifficultyHard, last: true}, true
	case eventVeryHardLastOptions[option]:
		return eventQuestSelector{difficulty: eventQuestDifficultyVeryHard, last: true}, true
	case eventAllQuestOptions[option]:
		return eventQuestSelector{all: true}, true
	case eventNormalFirstOptions[option]:
		return eventQuestSelector{difficulty: eventQuestDifficultyNormal, ordinal: 1}, true
	case option == 665:
		return eventQuestSelector{difficulty: eventQuestDifficultyNormal, ordinal: 5}, true
	case eventExHardFifthOptions[option]:
		return eventQuestSelector{difficulty: eventQuestDifficultyExHard, ordinal: 5}, true
	case option == 697:
		return eventQuestSelector{difficulty: eventQuestDifficultyExHard, ordinal: 3}, true
	case option == 200018:
		return eventQuestSelector{difficulty: eventQuestDifficultyExHard, ordinal: 6}, true
	case option >= 519 && option <= 521:
		return eventQuestSelector{sortOrder: option - 518}, true
	default:
		return eventQuestSelector{}, false
	}
}

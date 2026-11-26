package main

// Explicit Keep selections from the completed 26 November channel workbook. Existing Australian channel IDs and numbers
// are preserved. Expansion uses stable names and numbers, never fuzzy matches.
var selectedCategories = map[string]string{
	"95": "General / FTA",
	"44": "Community / Public",
	"1209": "News / Business",
	"1210": "News / Business",
	"1202": "News / Business",
	"1200": "News / Business",
	"1207": "News / Business",
	"1203": "News / Business",
	"1206": "News / Business",
	"1204": "News / Business",
	"1201": "News / Business",
	"1101": "Science / Tech / Nature",
	"1100": "Science / Tech / Nature",
	"1410": "Science / Tech / Nature",
	"1600": "Drama / Series",
	"1601": "Drama / Series",
	"1602": "Drama / Series",
	"1603": "Drama / Series",
	"1502": "Comedy",
	"1500": "Comedy",
	"1503": "Crime / Mystery",
	"1504": "Crime / Mystery",
	"1604": "Crime / Mystery",
	"1507": "Movies",
	"1700": "Movies",
	"1508": "Movies",
	"1710": "Kids / Family",
	"1720": "Food / Lifestyle",
	"1509": "Food / Lifestyle",
	"1510": "Food / Lifestyle",
	"1721": "Food / Lifestyle",
	"1722": "Food / Lifestyle",
	"1730": "Reality / Entertainment",
	"1301": "Sport",
	"1300": "Sport",
	"1306": "Sport",
	"1304": "Sport",
}

func expandedCategory(ch Channel) string {
	for _, spec:=range extraAustralian {
		if ch.Name==spec.name && ch.Number==spec.number && (ch.Network==spec.net || (spec.net=="" && ch.Network=="International")) {
			return selectedCategories[spec.number]
		}
	}
	return ""
}

func channelGroup(ch Channel) string {
	if ch.Category!="" {return ch.Category}
	return ch.Network
}

// Expanded streams are always inspected through the guarded relay rather
// than redirecting Jellyfin directly to a provider. No change to Fast Tune.
func requiresGuardedRelay(ch Channel) bool {
	return installedProviders.IDs[ch.ID] || (extraAustralianEnabled && expandedCategory(ch)!="")
}

package main

// Round 2 approved shortlist. Existing Australian channel IDs and numbers
// are preserved. Expansion uses stable names and numbers, never fuzzy matches.
var roundTwoCategories = map[string]string{
	"95": "General & community",
	"44": "General & community",
	"1000": "General & community",
	"1001": "General & community",
	"1200": "News & business",
	"1201": "News & business",
	"1202": "News & business",
	"1203": "News & business",
	"1204": "News & business",
	"1205": "News & business",
	"1206": "News & business",
	"1207": "News & business",
	"1208": "News & business",
	"1300": "Sports",
	"1301": "Sports",
	"1302": "Sports",
	"1303": "Sports",
	"1304": "Sports",
	"1305": "Sports",
	"1306": "Sports",
	"1307": "Sports",
	"1400": "Factual & documentaries",
	"1401": "Factual & documentaries",
	"1402": "Factual & documentaries",
	"1403": "Factual & documentaries",
	"1404": "Factual & documentaries",
	"1500": "Multi-show entertainment",
	"1501": "Multi-show entertainment",
	"1502": "Multi-show entertainment",
	"1503": "Multi-show entertainment",
	"1504": "Multi-show entertainment",
	"1505": "Multi-show entertainment",
	"1506": "Multi-show entertainment",
	"1507": "Multi-show entertainment",
	"1508": "Multi-show entertainment",
	"1509": "Multi-show entertainment",
	"1510": "Multi-show entertainment",
	"1511": "Multi-show entertainment",
}

func expandedCategory(ch Channel) string {
	for _, spec:=range extraAustralian {
		if ch.Name==spec.name && ch.Number==spec.number && (ch.Network==spec.net || (spec.net=="" && ch.Network=="International")) {
			return roundTwoCategories[spec.number]
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

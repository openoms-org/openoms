package olx

// transactionStatusMapping maps OLX transaction statuses to OpenOMS order statuses.
var transactionStatusMapping = map[string]string{
	"pending":   "new",
	"completed": "confirmed",
	"paid":      "confirmed",
	"cancelled": "cancelled",
}

// MapTransactionStatus translates an OLX transaction status to an OpenOMS order status.
func MapTransactionStatus(olxStatus string) (openomsStatus string, ok bool) {
	openomsStatus, ok = transactionStatusMapping[olxStatus]
	return
}

// advertStatusMapping maps OLX advert statuses to OpenOMS listing sync statuses.
var advertStatusMapping = map[string]string{
	"new":                  "pending",
	"active":               "synced",
	"limited":              "error",
	"outdated":             "inactive",
	"removed_by_user":      "inactive",
	"removed_by_moderator": "error",
	"moderated":            "pending",
	"blocked":              "error",
	"disabled":             "error",
	"unconfirmed":          "pending",
	"unpaid":               "pending",
}

// MapAdvertStatus translates an OLX advert status to an OpenOMS listing sync status.
func MapAdvertStatus(olxStatus string) (syncStatus string, ok bool) {
	syncStatus, ok = advertStatusMapping[olxStatus]
	return
}

// Command ouigen regenerates oui_table.go from the IEEE MA-L registry.
//
// It downloads the registry CSV, keeps the assignments of a fixed allowlist of
// well-known vendors, takes the lowest few OUIs of each, and writes them as a
// sorted Go array. Every allowlisted vendor must match at least one row so
// registry renames are noticed.
package main

import (
	"bytes"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
)

const defaultURL = "https://standards-oui.ieee.org/oui/oui.csv"

// vendor pairs a display name with an anchored pattern over the registry's
// exact "Organization Name" column.
type vendor struct {
	match *regexp.Regexp
	name  string
}

func v(name, pattern string) vendor {
	return vendor{name: name, match: regexp.MustCompile("^(?:" + pattern + ")$")}
}

// vendors is the allowlist, matched against the exact organization names in
// the registry. Patterns are anchored, so an unrelated company that merely
// contains a vendor's name never matches.
var vendors = []vendor{
	v("AMD", `AMD|Advanced Micro Devices, Inc\.`),
	v("ASUS", `ASUSTek COMPUTER INC\.`),
	v("AVM", `AVM Audiovisuelles Marketing und Computersysteme GmbH|AVM GmbH`),
	v("Amazon", `Amazon Technologies Inc\.`),
	v("Apple", `Apple, Inc\.`),
	v("Arduino", `ARDUINO AG`),
	v("Arista", `Arista Networks|Arista Networks, Inc\.`),
	v("Bosch", `Robert Bosch GmbH`),
	v("Broadcom", `Broadcom|Broadcom Limited`),
	v("Brother", `Brother [Ii]ndustries, LTD\.`),
	v("Canon", `CANON INC\.`),
	v("Cisco", `Cisco Systems, Inc|Cisco Meraki`),
	v("Commscope", `Commscope`),
	v("D-Link", `D-Link International|D-Link Corporation`),
	v("DJI", `DJI BAIWANG TECHNOLOGY CO LTD`),
	v("Dell", `Dell Inc\.`),
	v("Epson", `Seiko Epson Corporation`),
	v("Ericsson", `Ericsson AB|Ericsson`),
	v("Espressif", `Espressif Inc\.`),
	v("Extreme Networks", `Extreme Networks, Inc\.`),
	v("Fitbit", `Fitbit, Inc\.`),
	v("Fortinet", `Fortinet, Inc\.`),
	v("Fujitsu", `FUJITSU LIMITED`),
	v("Garmin", `Garmin International`),
	v("Gigabyte", `GIGA-BYTE TECHNOLOGY CO\.,LTD\.`),
	v("Google", `Google, Inc\.`),
	v("GoPro", `GoPro`),
	v("H3C", `New H3C Technologies Co\., Ltd`),
	v("HP", `Hewlett Packard|HP Inc\.`),
	v("HPE", `Hewlett Packard Enterprise`),
	v("Hikvision", `Hangzhou Hikvision Digital Technology Co\.,Ltd\.`),
	v("Hisense", `HISENSE VISUAL TECHNOLOGY CO\.,LTD`),
	v("Hitachi", `Hitachi, Ltd\.`),
	v("Honeywell", `Honeywell`),
	v("Huawei", `HUAWEI TECHNOLOGIES CO\.,LTD|Huawei Device Co\., Ltd\.`),
	v("IBM", `IBM Corp|IBM`),
	v("Intel", `Intel Corporate|Intel Corporation`),
	v("Juniper", `Juniper Networks`),
	v("Konica Minolta", `KONICA MINOLTA HOLDINGS, INC\.`),
	v("Kyocera", `KYOCERA CORPORATION|KYOCERA Corporation`),
	v("LG", `LG Electronics \(Mobile Communications\)|LG Electronics`),
	v("Lenovo", `Lenovo Mobile Communication Technology Ltd\.|Lenovo`),
	v("Linksys", `Cisco-Linksys, LLC`),
	v("Logitech", `Logitech|Logitech, Inc`),
	v("MSI", `Micro-Star INTL CO\., LTD\.`),
	v("Marvell", `Marvell Semiconductors`),
	v("MediaTek", `MediaTek Inc`),
	v("Mellanox", `Mellanox Technologies, Inc\.`),
	v("Meta", `Meta Platforms, Inc\.`),
	v("Microsoft", `Microsoft Corporation`),
	v("MikroTik", `Routerboard\.com`),
	v("Motorola", `Motorola Mobility LLC, a Lenovo Company`),
	v("NEC", `NEC Platforms, Ltd\.|NEC Corporation`),
	v("NETGEAR", `NETGEAR`),
	v("NVIDIA", `NVIDIA Corporation`),
	v("NXP", `NXP Semiconductors|NXP Semiconductor \(Tianjin\) LTD\.`),
	v("Nintendo", `Nintendo Co\.,Ltd|Nintendo Co\., Ltd\.`),
	v("Nokia", `Nokia|Nokia Corporation`),
	v("Nordic Semiconductor", `Nordic Semiconductor ASA`),
	v("OPPO", `GUANGDONG OPPO MOBILE TELECOMMUNICATIONS CORP\.,LTD`),
	v("Oracle", `Oracle Corporation`),
	v("Palo Alto Networks", `Palo Alto Networks`),
	v("Panasonic", `Panasonic Corporation AVC Networks Company`),
	v("Philips", `Philips`),
	v("QNAP", `QNAP Systems, Inc\.`),
	v("Qualcomm", `Qualcomm Inc\.`),
	v("Raspberry Pi", `Raspberry Pi Trading Ltd|Raspberry Pi \(Trading\) Ltd|Raspberry Pi Foundation`),
	v("Realtek", `REALTEK SEMICONDUCTOR CORP\.`),
	v("Rockwell Automation", `Rockwell Automation`),
	v("Roku", `Roku, Inc\.?`),
	v("Ruckus", `Ruckus Wireless`),
	v("STMicroelectronics", `STMicro(?:e)?lectronics International NV`),
	v("Sagemcom", `Sagemcom Broadband SAS`),
	v("Samsung", `Samsung Electronics Co\.,Ltd`),
	v("Schneider Electric", `Schneider Electric`),
	v("Sharp", `SHARP Corporation`),
	v("Siemens", `Siemens AG|SIEMENS AG`),
	v("Silicon Labs", `Silicon Laboratories`),
	v("Sonos", `Sonos, Inc\.`),
	v("Sony", `Sony Corporation|Sony Interactive Entertainment Inc\.`),
	v("Super Micro", `Super Micro Computer, Inc\.`),
	v("Synology", `Synology Incorporated`),
	v("TCL", `TCL King Electrical Appliances\(Huizhou\)Co\.,Ltd`),
	v("TP-Link", `TP-LINK TECHNOLOGIES CO\.,LTD\.|TP-Link Systems Inc\.?`),
	v("Tesla", `Tesla,Inc\.`),
	v("Texas Instruments", `Texas Instruments`),
	v("Toshiba", `Toshiba`),
	v("Ubiquiti", `Ubiquiti Inc`),
	v("VMware", `VMware, Inc\.`),
	v("Vizio", `Vizio, Inc`),
	v("Xerox", `XEROX CORPORATION|Xerox Corporation`),
	v("Xiaomi", `Xiaomi Communications Co Ltd`),
	v("ZTE", `zte corporation`),
	v("Zebra", `Zebra Technologies Inc\.?`),
	v("eero", `eero inc\.`),
	v("vivo", `vivo Mobile Communication Co\., Ltd\.`),
}

// entry is one selected assignment.
type entry struct {
	vendor string
	org    string
	oui    [3]byte
}

func main() {
	var (
		output    = flag.String("o", "oui_table.go", "output file")
		url       = flag.String("url", defaultURL, "IEEE MA-L CSV URL")
		perVendor = flag.Int("per-vendor", 3, "assignments kept per vendor")
	)
	flag.Parse()
	if err := run(*output, *url, *perVendor); err != nil {
		fmt.Fprintln(os.Stderr, "ouigen:", err)
		os.Exit(1)
	}
}

func run(output, url string, perVendor int) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	// The IEEE registry answers 418 to Go's default User-Agent.
	req.Header.Set("User-Agent", "curl/8.0 (randomizer-go ouigen)")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch %s: %s", url, resp.Status)
	}
	records, err := readRegistry(resp.Body)
	if err != nil {
		return err
	}
	entries, err := selectOUIs(records, vendors, perVendor)
	if err != nil {
		return err
	}
	src, err := render(entries, url)
	if err != nil {
		return err
	}
	return os.WriteFile(output, src, 0o644)
}

// readRegistry parses the registry CSV, returning the MA-L rows as
// (assignment, organization) pairs.
func readRegistry(r io.Reader) ([][2]string, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true
	header, err := reader.Read()
	if err != nil {
		return nil, err
	}
	if len(header) < 3 || header[0] != "Registry" || header[1] != "Assignment" || header[2] != "Organization Name" {
		return nil, fmt.Errorf("unexpected header %q", header)
	}
	var rows [][2]string
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			return rows, nil
		}
		if err != nil {
			return nil, err
		}
		if len(record) < 3 || record[0] != "MA-L" {
			continue
		}
		rows = append(rows, [2]string{strings.ToUpper(record[1]), strings.TrimSpace(record[2])})
	}
}

// selectOUIs keeps, for every vendor, the perVendor lowest universal unicast
// assignments whose organization matches, sorted by value.
func selectOUIs(rows [][2]string, vendors []vendor, perVendor int) ([]entry, error) {
	byVendor := make([][]entry, len(vendors))
	for _, row := range rows {
		for i, vend := range vendors {
			if !vend.match.MatchString(row[1]) {
				continue
			}
			var oui [3]byte
			if _, err := hex.Decode(oui[:], []byte(row[0])); err != nil || len(row[0]) != 6 {
				return nil, fmt.Errorf("assignment %q: bad OUI", row[0])
			}
			if oui[0]&0x03 == 0 {
				byVendor[i] = append(byVendor[i], entry{oui: oui, vendor: vend.name, org: row[1]})
			}
			break
		}
	}
	var entries []entry
	for i, vend := range vendors {
		list := byVendor[i]
		if len(list) == 0 {
			return nil, fmt.Errorf("vendor %q matched no registry row", vend.name)
		}
		sort.Slice(list, func(a, b int) bool { return bytes.Compare(list[a].oui[:], list[b].oui[:]) < 0 })
		entries = append(entries, list[:min(perVendor, len(list))]...)
	}
	sort.Slice(entries, func(a, b int) bool { return bytes.Compare(entries[a].oui[:], entries[b].oui[:]) < 0 })
	return entries, nil
}

// render formats entries as the Go source of oui_table.go.
func render(entries []entry, url string) ([]byte, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "// Code generated by internal/ouigen from %s; DO NOT EDIT.\n\n", url)
	b.WriteString("package randomizer\n\n")
	b.WriteString("// realOUIs are universal unicast OUIs of well-known vendors from the IEEE MA-L registry, sorted ascending.\n")
	b.WriteString("var realOUIs = [...][3]byte{\n")
	for _, e := range entries {
		fmt.Fprintf(&b, "\t{0x%02x, 0x%02x, 0x%02x}, // %s (%s)\n", e.oui[0], e.oui[1], e.oui[2], e.vendor, e.org)
	}
	b.WriteString("}\n")
	return format.Source([]byte(b.String()))
}

package settings

import (
	"maps"
	"slices"
)

// ClonePdfGlobal returns an independent snapshot of src. Slices and maps
// (Allow, Network*, FontPaths, ExcludeFromOutline, Header/Footer.Replace)
// are copied so later mutations of src cannot affect the result.
func ClonePdfGlobal(src PdfGlobal) PdfGlobal {
	dst := src
	dst.Header = CloneHeaderFooter(src.Header)
	dst.Footer = CloneHeaderFooter(src.Footer)
	dst.Load.Allow = cloneStrings(src.Load.Allow)
	dst.Load.NetworkAllowedSchemes = cloneStrings(src.Load.NetworkAllowedSchemes)
	dst.Load.NetworkAllowedHosts = cloneStrings(src.Load.NetworkAllowedHosts)
	dst.ExcludeFromOutline = cloneStrings(src.ExcludeFromOutline)
	dst.FontPaths = cloneStrings(src.FontPaths)

	return dst
}

// ClonePdfObject returns an independent snapshot of src, including Header/
// Footer.Replace, load maps, and InlineHTML.
func ClonePdfObject(src PdfObject) PdfObject {
	dst := src
	dst.Header = CloneHeaderFooter(src.Header)
	dst.Footer = CloneHeaderFooter(src.Footer)
	dst.Load.CustomHeaders = cloneStringMap(src.Load.CustomHeaders)
	dst.Load.InlineHTML = cloneBytes(src.Load.InlineHTML)

	return dst
}

// CloneImageGlobal returns an independent snapshot of src, including Allow,
// and Network* slices.
func CloneImageGlobal(src ImageGlobal) ImageGlobal {
	dst := src
	dst.Load.Allow = cloneStrings(src.Load.Allow)
	dst.Load.NetworkAllowedSchemes = cloneStrings(src.Load.NetworkAllowedSchemes)
	dst.Load.NetworkAllowedHosts = cloneStrings(src.Load.NetworkAllowedHosts)

	return dst
}

// CloneHeaderFooter returns an independent snapshot of src, including Replace.
func CloneHeaderFooter(src HeaderFooter) HeaderFooter {
	dst := src
	dst.Replace = cloneStringMap(src.Replace)

	return dst
}

func cloneStrings(src []string) []string {
	return slices.Clone(src)
}

func cloneBytes(src []byte) []byte {
	return slices.Clone(src)
}

func cloneStringMap(source map[string]string) map[string]string {
	return maps.Clone(source)
}

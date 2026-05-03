package utils

import (
	"strings"
)

var (
	adDomains = []string{
		"doubleclick.net",
		"googlesyndication.com",
		"googleadservices.com",
		"google-analytics.com",
		"googletagmanager.com",
		"googletagservices.com",
		"adservice.google.com",
		"pagead2.googlesyndication.com",
		"adsense",
		"adnxs.com",
		"adsrvr.org",
		"adform.net",
		"criteo.com",
		"criteo.net",
		"facebook.com/tr",
		"connect.facebook.net",
		"amazon-adsystem.com",
		"ads.yahoo.com",
		"advertising.com",
		"taboola.com",
		"outbrain.com",
		"mgid.com",
		"revcontent.com",
		"propellerads.com",
		"popcash.net",
		"popads.net",
		"exoclick.com",
		"juicyads.com",
		"trafficjunky.com",
		"adsterra.com",

		"tracking.",
		"tracker.",
		"analytics.",
		"pixel.",
		"beacon.",
		"telemetry.",
		"metrics.",
		"stats.",
		"log.",
		"collect.",

		"iklan",
		"sponsor",

		"popunder",
		"popup",
		"redirect",
		"redir.",
	}

	adPathPatterns = []string{
		"/ads/",
		"/ad/",
		"/Ads/",
		"/Ad/",
		"/advertisement/",
		"/advert/",
		"/banner/",
		"/banners/",
		"/sponsor/",
		"/tracking/",
		"/tracker/",
		"/pixel/",
		"/beacon/",
		"/analytics/",
		"/telemetry/",
		"/log/",
		"/collect/",
		"/promo/",
		"/campaign/",
		"ad=",
		"ads=",
		"adid=",
		"ad_",
		"_ad.",
		"-ad-",
		"-ads-",
	}

	adFilePatterns = []string{
		"ad_video",
		"ads_video",
		"advertisement",
		"promo_",
		"sponsor_",
		"banner_",
		"preroll",
		"midroll",
		"postroll",
		"interstitial",
	}

	trustedVideoDomains = []string{
		"filedon.co",
		"pixeldrain.com",
		"acefile.co",
		"drive.google.com",
		"drive.usercontent.google.com",
		"googleusercontent.com",
		"mixdrop.co",
		"mixdrop.to",
		"mixdrop.ch",
		"m1xdrop.net",
		"mega.nz",
		"mediafire.com",
		"zippyshare.com",
		"uploaded.net",
		"rapidgator.net",
	}
)

func IsAdURL(url string) bool {
	lowerURL := strings.ToLower(url)

	for _, domain := range adDomains {
		if strings.Contains(lowerURL, domain) {
			return true
		}
	}

	for _, pattern := range adPathPatterns {
		if strings.Contains(lowerURL, strings.ToLower(pattern)) {
			return true
		}
	}

	for _, pattern := range adFilePatterns {
		if strings.Contains(lowerURL, pattern) {
			return true
		}
	}

	return false
}

func IsLikelyVideoURL(url string) bool {
	lowerURL := strings.ToLower(url)

	videoExtensions := []string{".mp4", ".mkv", ".avi", ".webm", ".m3u8", ".ts", ".flv", ".mov"}
	hasVideoExt := false
	for _, ext := range videoExtensions {
		if strings.Contains(lowerURL, ext) {
			hasVideoExt = true
			break
		}
	}

	if !hasVideoExt {
		return false
	}

	return !IsAdURL(url)
}

func IsTrustedDomain(url string) bool {
	lowerURL := strings.ToLower(url)
	for _, domain := range trustedVideoDomains {
		if strings.Contains(lowerURL, domain) {
			return true
		}
	}
	return false
}

func FilterVideoURL(url string) bool {
	if url == "" || IsAdURL(url) {
		return false
	}

	lowerURL := strings.ToLower(url)

	isMedia := strings.Contains(lowerURL, ".mp4") ||
		strings.Contains(lowerURL, ".mkv") ||
		strings.Contains(lowerURL, ".m3u8") ||
		strings.Contains(lowerURL, ".ts") ||
		strings.Contains(lowerURL, "download=") ||
		strings.Contains(lowerURL, "export=download") ||
		strings.Contains(lowerURL, "&confirm=") ||
		strings.Contains(lowerURL, "doc-") ||
		strings.Contains(lowerURL, "googleusercontent.com/download")

	if !isMedia {
		return false
	}

	return IsTrustedDomain(url) || IsLikelyVideoURL(url)
}

func SanitizeVideoURL(url string) string {
	return url
}

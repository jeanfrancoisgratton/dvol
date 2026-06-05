// dvol
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/08/25 00:16
// Original filename: src/types/errorcodes.go

package types

// shellExitCode maps your internal error code space (100s, 200s, …) to small, shell-friendly exits.
// Rationale:
//

// 100–199  JSON-related              -> 10
// 200–299  ImagePull                 -> 20
// 300–399  Container start/stop/etc. -> 30
// 400–499  Container archive         -> 40
// 500–599  API version               -> 50
// 600–699  Volume ops                -> 60
// 700–799  Tarball read/decompress   -> 70
// 800–899  Restore request handling  -> 80
// <100     keep as-is (your early small codes, e.g., 10–12 in main.go)
// other    fallback to 1
func ShellExitCode(internal int) int {
	switch {
	case internal < 0:
		return 1
	case internal < 100:
		// Keep your small preflight errors (e.g., 10–12) as-is; they're already shell-safe.
		return internal
	case internal < 200:
		return 10
	case internal < 300:
		return 20
	case internal < 400:
		return 30
	case internal < 500:
		return 40
	case internal < 600:
		return 50
	case internal < 700:
		return 60
	case internal < 800:
		return 70
	case internal < 900:
		return 80
	default:
		// Unknown/new categories: collapse to 1 rather than risk modulo surprises.
		return 1
	}
}

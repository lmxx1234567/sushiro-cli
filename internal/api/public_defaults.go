package api

// Retained at the user's explicit request from Ryujoxys/sushiro-overdose,
// revision e273df046789773616c7851c0bea14d4546f47e5, internal/app/queue_live.go.
// This is an upstream public-query compatibility default, not a personal
// session or an official anonymous bootstrap. Issuer, distribution conditions
// and lifetime are not established. See THIRD_PARTY_NOTICES.md.
const upstreamPublicQueryAuthorization = "4OI44O844Kv44Oz5qSc6Ki855So77yad2VjaGF05YWx6YCa4"

// DefaultPublicConfig returns an in-memory fallback only. It is never saved by
// the service. Explicit configurations, including invalid ones, take precedence.
func DefaultPublicConfig(profile string) PublicConfig {
	return PublicConfig{SchemaVersion: 1, Profile: profile, BaseURL: BaseURL,
		QueryAuthorization: upstreamPublicQueryAuthorization}
}

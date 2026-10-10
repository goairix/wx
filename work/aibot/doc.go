// Package aibot implements the enterprise intelligent-bot WebSocket protocol.
//
// Bot credentials are independent of enterprise application access tokens.
// Connect authenticates and starts the lifecycle; its context stays in effect
// across reconnects. Drain Notifications continuously and dispatch business
// work separately. Close permanently ends the client and outstanding requests.
// Outbound messages are never replayed automatically after connection loss.
//
// The caller owns replica leases, deduplication, sessions, and agent execution.
package aibot

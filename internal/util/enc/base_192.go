package enc

// TODO: Base 192 would be even more optimal than Base 128.
// Base192 could take the top 192 characters (leaving out the bottom 32, which are usually control characters).
// Base192 encoding could achieve near-raw efficiency, as it encodes 7.5 bits / byte. Or, in other words: every
// 15 bytes get encoded into 16 octets. This yields an appropriate 6.66% encoding loss.

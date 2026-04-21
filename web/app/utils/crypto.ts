import { sha512 } from '@noble/hashes/sha2.js'
import * as ed from '@noble/ed25519'

// Configure ed25519 to use noble-hashes sha512 (v3.x style)
ed.hashes.sha512 = (m: Uint8Array) => sha512(m)
ed.hashes.sha512Async = (m: Uint8Array) => Promise.resolve(sha512(m))

// Fix for environments where crypto.webcrypto is missing or different
if (typeof window !== 'undefined') {
  if (!globalThis.crypto) (globalThis as any).crypto = window.crypto
}

export const extractPrivateKeyFromPEM = (pem: string): Uint8Array => {
  const clean = pem.replace(/-----(BEGIN|END) (EC|ED25519|PRIVATE) KEY-----/g, '').replace(/\s/g, '')
  const der = Uint8Array.from(atob(clean), c => c.charCodeAt(0))
  
  // Ed25519 PKCS8 detection: Header: 30 2e 02 01 00 30 05 06 03 2b 65 70 04 22 04 20
  if (der[0] === 0x30 && der[7] === 0x06 && der[8] === 0x03 && der[9] === 0x2b && der[10] === 0x65 && der[11] === 0x70) {
    return der.slice(16, 48)
  }
  
  // Fallback: assume Ed25519 for simple 32-byte chunks or default
  return der.slice(-32)
}

export const signMessage = async (message: string, privKeyPEM: string): Promise<string> => {
  const privKey = extractPrivateKeyFromPEM(privKeyPEM)
  const messageBytes = new TextEncoder().encode(message)
  
  const signature = await ed.sign(messageBytes, privKey)
  
  return btoa(Array.from(signature).map(b => String.fromCharCode(b)).join(''))
}

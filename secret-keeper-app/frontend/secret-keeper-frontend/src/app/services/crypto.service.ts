import { Injectable } from '@angular/core';

@Injectable({ providedIn: 'root' })
export class CryptoService {

  generateRoomKey(): string {
    const bytes = window.crypto.getRandomValues(new Uint8Array(18));
    return this.bytesToBase64(bytes);
  }

  async deriveConversationKey(passphrase: string, convId: string): Promise<CryptoKey> {
    const enc = new TextEncoder();
    const keyMaterial = await window.crypto.subtle.importKey(
      'raw', enc.encode(passphrase), 'PBKDF2', false, ['deriveKey']
    );
    return window.crypto.subtle.deriveKey(
      { name: 'PBKDF2', salt: enc.encode(convId), iterations: 100000, hash: 'SHA-256' },
      keyMaterial,
      { name: 'AES-GCM', length: 256 },
      false,
      ['encrypt', 'decrypt']
    );
  }

  async encryptMessage(plaintext: string, convKey: CryptoKey): Promise<string> {
    const iv = window.crypto.getRandomValues(new Uint8Array(12));
    const enc = new TextEncoder();
    const ciphertext = await window.crypto.subtle.encrypt(
      { name: 'AES-GCM', iv }, convKey, enc.encode(plaintext)
    );
    const combined = new Uint8Array(iv.length + ciphertext.byteLength);
    combined.set(iv, 0);
    combined.set(new Uint8Array(ciphertext), iv.length);
    return this.bytesToBase64(combined);
  }

  async decryptMessage(encryptedB64: string, convKey: CryptoKey): Promise<string> {
    const combinedBuffer = this.base64ToArrayBuffer(encryptedB64);
    const combined = new Uint8Array(combinedBuffer);
    const iv = combined.slice(0, 12);
    const ciphertext = combined.slice(12);
    const plaintext = await window.crypto.subtle.decrypt(
      { name: 'AES-GCM', iv }, convKey, ciphertext
    );
    return new TextDecoder().decode(plaintext);
  }

  // ── RSA key pair ────────────────────────────────────────────────────────────

  async generateRsaKeyPair(): Promise<CryptoKeyPair> {
    return window.crypto.subtle.generateKey(
      { name: 'RSA-OAEP', modulusLength: 2048, publicExponent: new Uint8Array([1, 0, 1]), hash: 'SHA-256' },
      true,
      ['encrypt', 'decrypt']
    );
  }

  async exportPublicKey(key: CryptoKey): Promise<string> {
    const exported = await window.crypto.subtle.exportKey('spki', key);
    return this.bytesToBase64(new Uint8Array(exported));
  }

  async importPublicKey(b64: string): Promise<CryptoKey> {
    const buf = this.base64ToArrayBuffer(b64);
    return window.crypto.subtle.importKey(
      'spki', buf,
      { name: 'RSA-OAEP', hash: 'SHA-256' },
      false, ['encrypt']
    );
  }

  async importPrivateKey(b64: string): Promise<CryptoKey> {
    const buf = this.base64ToArrayBuffer(b64);
    return window.crypto.subtle.importKey(
      'pkcs8', buf,
      { name: 'RSA-OAEP', hash: 'SHA-256' },
      false, ['decrypt']
    );
  }

  /** Encrypt a short string (e.g. room key passphrase) with an RSA public key. */
  async rsaEncrypt(plaintext: string, publicKey: CryptoKey): Promise<string> {
    const enc = new TextEncoder();
    const cipher = await window.crypto.subtle.encrypt(
      { name: 'RSA-OAEP' }, publicKey, enc.encode(plaintext)
    );
    return this.bytesToBase64(new Uint8Array(cipher));
  }

  /** Decrypt an RSA-encrypted ciphertext with a private key. */
  async rsaDecrypt(ciphertextB64: string, privateKey: CryptoKey): Promise<string> {
    const buf = this.base64ToArrayBuffer(ciphertextB64);
    const plain = await window.crypto.subtle.decrypt(
      { name: 'RSA-OAEP' }, privateKey, buf
    );
    return new TextDecoder().decode(plain);
  }

  // ── Private key wrapping (AES-GCM, password-derived) ──────────────────────

  /** Derive an AES-GCM wrapping key from the user's password + username. */
  async deriveWrappingKey(password: string, username: string): Promise<CryptoKey> {
    const enc = new TextEncoder();
    const keyMaterial = await window.crypto.subtle.importKey(
      'raw', enc.encode(password), 'PBKDF2', false, ['deriveKey']
    );
    return window.crypto.subtle.deriveKey(
      { name: 'PBKDF2', salt: enc.encode('sk-wrap-' + username), iterations: 100000, hash: 'SHA-256' },
      keyMaterial,
      { name: 'AES-GCM', length: 256 },
      false,
      ['wrapKey', 'unwrapKey']
    );
  }

  /** Export + AES-GCM wrap the private key; returns "ivB64:wrappedB64". */
  async wrapPrivateKey(privateKey: CryptoKey, wrappingKey: CryptoKey): Promise<string> {
    const iv = window.crypto.getRandomValues(new Uint8Array(12));
    const wrapped = await window.crypto.subtle.wrapKey(
      'pkcs8', privateKey, wrappingKey, { name: 'AES-GCM', iv }
    );
    return this.bytesToBase64(iv) + ':' + this.bytesToBase64(new Uint8Array(wrapped));
  }

  /** Unwrap a private key from "ivB64:wrappedB64" format. */
  async unwrapPrivateKey(wrapped: string, wrappingKey: CryptoKey): Promise<CryptoKey> {
    const [ivB64, dataB64] = wrapped.split(':');
    const ivBytes = this.base64ToBytes(ivB64);
    const iv = ivBytes.buffer.slice(ivBytes.byteOffset, ivBytes.byteOffset + ivBytes.byteLength) as ArrayBuffer;
    const data = this.base64ToArrayBuffer(dataB64) as ArrayBuffer;
    return window.crypto.subtle.unwrapKey(
      'pkcs8', data, wrappingKey,
      { name: 'AES-GCM', iv },
      { name: 'RSA-OAEP', hash: 'SHA-256' },
      true, ['decrypt']
    );
  }

  // ── Helpers ─────────────────────────────────────────────────────────────────

  bytesToBase64(bytes: Uint8Array): string {
    let binary = '';
    const chunkSize = 0x8000;
    for (let index = 0; index < bytes.length; index += chunkSize) {
      const chunk = bytes.subarray(index, index + chunkSize);
      for (let chunkIndex = 0; chunkIndex < chunk.length; chunkIndex += 1) {
        binary += String.fromCharCode(chunk[chunkIndex]);
      }
    }
    return btoa(binary);
  }

  base64ToBytes(base64: string): Uint8Array {
    return new Uint8Array(this.base64ToArrayBuffer(base64));
  }

  base64ToArrayBuffer(base64: string): ArrayBuffer {
    const binary = atob(base64);
    const buffer = new ArrayBuffer(binary.length);
    const bytes = new Uint8Array(buffer);
    for (let index = 0; index < binary.length; index += 1) {
      bytes[index] = binary.charCodeAt(index);
    }
    return buffer;
  }
}

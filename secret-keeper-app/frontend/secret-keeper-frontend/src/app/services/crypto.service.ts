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


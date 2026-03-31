import { TestBed } from '@angular/core/testing';
import { CryptoService } from './crypto.service';

describe('CryptoService', () => {
  let service: CryptoService;

  beforeEach(() => {
    TestBed.configureTestingModule({});
    service = TestBed.inject(CryptoService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  describe('generateRoomKey()', () => {
    it('should return a non-empty base64 string', () => {
      const key = service.generateRoomKey();
      expect(typeof key).toBe('string');
      expect(key.length).toBeGreaterThan(0);
    });

    it('should return a different key on each call', () => {
      const key1 = service.generateRoomKey();
      const key2 = service.generateRoomKey();
      expect(key1).not.toBe(key2);
    });

    it('should produce a valid base64-decodable string of 18 raw bytes', () => {
      const key = service.generateRoomKey();
      const decoded = atob(key);
      expect(decoded.length).toBe(18);
    });
  });

  describe('deriveConversationKey()', () => {
    it('should return a CryptoKey', async () => {
      const key = await service.deriveConversationKey('mypassphrase', 'conv-id-123');
      expect(key).toBeTruthy();
      expect(key.type).toBe('secret');
    });

    it('should produce the same key for the same passphrase and convId', async () => {
      const key1 = await service.deriveConversationKey('same-pass', 'same-conv');
      const key2 = await service.deriveConversationKey('same-pass', 'same-conv');
      const encrypted = await service.encryptMessage('hello', key1);
      const decrypted = await service.decryptMessage(encrypted, key2);
      expect(decrypted).toBe('hello');
    });

    it('should produce different keys for different passphrases', async () => {
      const key1 = await service.deriveConversationKey('pass-a', 'conv-1');
      const key2 = await service.deriveConversationKey('pass-b', 'conv-1');
      const encrypted = await service.encryptMessage('hello', key1);
      await expect(service.decryptMessage(encrypted, key2)).rejects.toThrow();
    });

    it('should produce different keys for different conversation IDs', async () => {
      const key1 = await service.deriveConversationKey('same-pass', 'conv-1');
      const key2 = await service.deriveConversationKey('same-pass', 'conv-2');
      const encrypted = await service.encryptMessage('hello', key1);
      await expect(service.decryptMessage(encrypted, key2)).rejects.toThrow();
    });
  });

  describe('encryptMessage() and decryptMessage()', () => {
    let key: CryptoKey;

    beforeEach(async () => {
      key = await service.deriveConversationKey('test-passphrase', 'test-conv-id');
    });

    it('should encrypt a message to a non-empty base64 string', async () => {
      const ciphertext = await service.encryptMessage('hello world', key);
      expect(typeof ciphertext).toBe('string');
      expect(ciphertext.length).toBeGreaterThan(0);
      expect(ciphertext).not.toBe('hello world');
    });

    it('should decrypt back to the original plaintext', async () => {
      const plaintext = 'secret message';
      const ciphertext = await service.encryptMessage(plaintext, key);
      const decrypted = await service.decryptMessage(ciphertext, key);
      expect(decrypted).toBe(plaintext);
    });

    it('should produce different ciphertexts for the same message (random IV)', async () => {
      const c1 = await service.encryptMessage('same message', key);
      const c2 = await service.encryptMessage('same message', key);
      expect(c1).not.toBe(c2);
    });

    it('should encrypt and decrypt an empty string', async () => {
      const ciphertext = await service.encryptMessage('', key);
      const decrypted = await service.decryptMessage(ciphertext, key);
      expect(decrypted).toBe('');
    });

    it('should encrypt and decrypt a long unicode message', async () => {
      // Covers 2-byte (Latin), 3-byte (CJK), and 4-byte (emoji/surrogate pair) UTF-8 sequences
      const long = '\u{1F512}\u3053\u3093\u306B\u3061\u306F\u{1F30D}'.repeat(100);
      const ciphertext = await service.encryptMessage(long, key);
      const decrypted = await service.decryptMessage(ciphertext, key);
      expect(decrypted).toBe(long);
    });

    it('should reject decryption when the ciphertext is tampered with', async () => {
      const ciphertext = await service.encryptMessage('hello', key);
      const bytes = Uint8Array.from(atob(ciphertext), c => c.charCodeAt(0));
      bytes[bytes.length - 1] ^= 0xff;
      const tampered = btoa(String.fromCharCode(...bytes));
      await expect(service.decryptMessage(tampered, key)).rejects.toThrow();
    });

    it('should reject decryption when the wrong key is used', async () => {
      const wrongKey = await service.deriveConversationKey('wrong-pass', 'test-conv-id');
      const ciphertext = await service.encryptMessage('hello', key);
      await expect(service.decryptMessage(ciphertext, wrongKey)).rejects.toThrow();
    });
  });
});

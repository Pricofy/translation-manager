import { LambdaClient, InvokeCommand } from '@aws-sdk/client-lambda';
import { fromIni } from '@aws-sdk/credential-provider-ini';

const FUNCTION_NAME = 'pricofy-translation-manager';
const REGION = process.env.AWS_REGION || 'eu-west-1';
const PROFILE = process.env.AWS_PROFILE || 'pricofy-dev';

const lambda = new LambdaClient({
  region: REGION,
  credentials: fromIni({ profile: PROFILE }),
});

interface TranslationRequest {
  texts: string[];
  sourceLang: string;
  targetLang: string;
}

interface TranslationResponse {
  translations?: string[];
  chunksProcessed?: number;
  error?: string;
}

async function invokeManager(request: TranslationRequest): Promise<TranslationResponse> {
  const command = new InvokeCommand({
    FunctionName: FUNCTION_NAME,
    Payload: Buffer.from(JSON.stringify(request)),
  });

  const result = await lambda.send(command);

  if (result.FunctionError) {
    throw new Error(`Lambda error: ${result.FunctionError}`);
  }

  const payload = Buffer.from(result.Payload!).toString();
  return JSON.parse(payload);
}

describe('Translation Manager E2E', () => {
  jest.setTimeout(60000);

  describe('Basic Translation', () => {
    it('should translate single text es→en', async () => {
      const response = await invokeManager({
        texts: ['Hola mundo'],
        sourceLang: 'es',
        targetLang: 'en',
      });

      expect(response.error).toBeUndefined();
      expect(response.translations).toHaveLength(1);
      expect(response.translations![0]).toBeTruthy();
      expect(response.chunksProcessed).toBe(1);
    });

    it('should translate batch of 10 texts', async () => {
      const response = await invokeManager({
        texts: Array.from({ length: 10 }, (_, i) => `Producto número ${i}`),
        sourceLang: 'es',
        targetLang: 'fr',
      });

      expect(response.error).toBeUndefined();
      expect(response.translations).toHaveLength(10);
      expect(response.chunksProcessed).toBe(1);
    });

    it('should handle empty texts array', async () => {
      const response = await invokeManager({
        texts: [],
        sourceLang: 'es',
        targetLang: 'fr',
      });

      expect(response.error).toBeUndefined();
      expect(response.translations).toEqual([]);
      expect(response.chunksProcessed).toBe(0);
    });
  });

  describe('Core Language Pairs (6 languages)', () => {
    const coreLanguages = ['es', 'fr', 'it', 'pt', 'de', 'en'];

    const sampleTexts: Record<string, string> = {
      es: 'Hola mundo',
      fr: 'Bonjour le monde',
      it: 'Ciao mondo',
      pt: 'Olá mundo',
      de: 'Hallo Welt',
      en: 'Hello world',
    };

    // Generate all 30 pairs
    const corePairs: [string, string][] = [];
    for (const source of coreLanguages) {
      for (const target of coreLanguages) {
        if (source !== target) {
          corePairs.push([source, target]);
        }
      }
    }

    test.each(corePairs)('should translate %s → %s', async (source, target) => {
      const response = await invokeManager({
        texts: [sampleTexts[source]],
        sourceLang: source,
        targetLang: target,
      });

      expect(response.error).toBeUndefined();
      expect(response.translations).toHaveLength(1);
      expect(response.translations![0]).toBeTruthy();
    });
  });

  describe('Extended Romance Languages', () => {
    const extendedPairs: [string, string, string][] = [
      ['ca', 'en', 'Bon dia'],
      ['en', 'ca', 'Good morning'],
      ['ro', 'en', 'Bună ziua'],
      ['gl', 'en', 'Bos días'],
      ['la', 'en', 'Salve'],
    ];

    test.each(extendedPairs)(
      'should translate %s → %s',
      async (source, target, text) => {
        const response = await invokeManager({
          texts: [text],
          sourceLang: source,
          targetLang: target,
        });

        expect(response.error).toBeUndefined();
        expect(response.translations).toHaveLength(1);
      }
    );
  });

  describe('Language Variants', () => {
    const variantPairs: [string, string, string][] = [
      ['es_MX', 'en', '¿Qué onda?'],
      ['en', 'es_MX', 'Hello friend'],
      ['pt_BR', 'en', 'Olá, tudo bem?'],
      ['en', 'pt_BR', 'Hello, how are you?'],
      ['fr_CA', 'en', 'Bonjour'],
    ];

    test.each(variantPairs)(
      'should translate %s → %s',
      async (source, target, text) => {
        const response = await invokeManager({
          texts: [text],
          sourceLang: source,
          targetLang: target,
        });

        expect(response.error).toBeUndefined();
        expect(response.translations).toHaveLength(1);
      }
    );
  });

  describe('Error Handling', () => {
    it('should return error for unsupported language', async () => {
      const response = await invokeManager({
        texts: ['Hello'],
        sourceLang: 'zh',
        targetLang: 'en',
      });

      expect(response.error).toBeTruthy();
    });

    it('should return error for same source and target', async () => {
      const response = await invokeManager({
        texts: ['Hello'],
        sourceLang: 'es',
        targetLang: 'es',
      });

      expect(response.error).toContain('must be different');
    });

    it('should return error for missing sourceLang', async () => {
      const response = await invokeManager({
        texts: ['Hello'],
        sourceLang: '',
        targetLang: 'fr',
      });

      expect(response.error).toContain('sourceLang');
    });

    it('should return error for missing targetLang', async () => {
      const response = await invokeManager({
        texts: ['Hello'],
        sourceLang: 'es',
        targetLang: '',
      });

      expect(response.error).toContain('targetLang');
    });
  });

  describe('Chunking (50 texts per chunk)', () => {
    it('should handle 50 texts in 1 chunk', async () => {
      const texts = Array.from({ length: 50 }, (_, i) => `Producto ${i}`);

      const response = await invokeManager({
        texts,
        sourceLang: 'es',
        targetLang: 'en',
      });

      expect(response.error).toBeUndefined();
      expect(response.translations).toHaveLength(50);
      expect(response.chunksProcessed).toBe(1);
    });

    it('should handle 100 texts in 2 chunks', async () => {
      const texts = Array.from({ length: 100 }, (_, i) => `Producto ${i}`);

      const response = await invokeManager({
        texts,
        sourceLang: 'es',
        targetLang: 'en',
      });

      expect(response.error).toBeUndefined();
      expect(response.translations).toHaveLength(100);
      expect(response.chunksProcessed).toBe(2);
    });

    it('should handle 150 texts in 3 chunks', async () => {
      const texts = Array.from({ length: 150 }, (_, i) => `Producto ${i}`);

      const response = await invokeManager({
        texts,
        sourceLang: 'es',
        targetLang: 'en',
      });

      expect(response.error).toBeUndefined();
      expect(response.translations).toHaveLength(150);
      expect(response.chunksProcessed).toBe(3);
    });

    it('should preserve text order across chunks', async () => {
      const texts = Array.from({ length: 100 }, (_, i) => `Texto número ${i}`);

      const response = await invokeManager({
        texts,
        sourceLang: 'es',
        targetLang: 'en',
      });

      expect(response.error).toBeUndefined();
      expect(response.translations).toHaveLength(100);

      // Verify order by checking numbers are preserved
      for (let i = 0; i < 100; i++) {
        expect(response.translations![i]).toContain(String(i));
      }
    });
  });

  describe('Pivot Translations (via EN)', () => {
    it('should handle ES→FR via EN pivot', async () => {
      const response = await invokeManager({
        texts: ['Buenos días amigo'],
        sourceLang: 'es',
        targetLang: 'fr',
      });

      expect(response.error).toBeUndefined();
      expect(response.translations).toHaveLength(1);
    });

    it('should handle DE→ES via EN pivot', async () => {
      const response = await invokeManager({
        texts: ['Guten Tag'],
        sourceLang: 'de',
        targetLang: 'es',
      });

      expect(response.error).toBeUndefined();
      expect(response.translations).toHaveLength(1);
    });

    it('should handle 50 texts with pivot', async () => {
      const texts = Array.from({ length: 50 }, (_, i) => `Produkt ${i}`);

      const response = await invokeManager({
        texts,
        sourceLang: 'de',
        targetLang: 'es',
      });

      expect(response.error).toBeUndefined();
      expect(response.translations).toHaveLength(50);
    });
  });

  describe('Edge Cases', () => {
    it('should handle texts with special characters', async () => {
      const response = await invokeManager({
        texts: ['¿Cómo estás? ¡Bien!', 'Ñoño señor', 'Prix: 100€'],
        sourceLang: 'es',
        targetLang: 'en',
      });

      expect(response.error).toBeUndefined();
      expect(response.translations).toHaveLength(3);
    });

    it('should handle empty strings in batch', async () => {
      const response = await invokeManager({
        texts: ['Hola', '', 'Mundo', ''],
        sourceLang: 'es',
        targetLang: 'en',
      });

      expect(response.error).toBeUndefined();
      expect(response.translations).toHaveLength(4);
      expect(response.translations![1]).toBe('');
      expect(response.translations![3]).toBe('');
    });
  });

  describe('Performance', () => {
    it('should complete 1 text in < 5s', async () => {
      const start = Date.now();

      await invokeManager({
        texts: ['Prueba rápida'],
        sourceLang: 'es',
        targetLang: 'en',
      });

      const duration = Date.now() - start;
      console.log(`1 text: ${duration}ms`);
      expect(duration).toBeLessThan(5000);
    });

    it('should complete 50 texts in < 10s', async () => {
      const texts = Array.from({ length: 50 }, (_, i) => `Producto ${i}`);
      const start = Date.now();

      await invokeManager({
        texts,
        sourceLang: 'es',
        targetLang: 'en',
      });

      const duration = Date.now() - start;
      console.log(`50 texts: ${duration}ms`);
      expect(duration).toBeLessThan(10000);
    });

    it('should complete 150 texts in < 30s', async () => {
      const texts = Array.from({ length: 150 }, (_, i) => `Producto ${i}`);
      const start = Date.now();

      await invokeManager({
        texts,
        sourceLang: 'es',
        targetLang: 'en',
      });

      const duration = Date.now() - start;
      console.log(`150 texts (3 chunks): ${duration}ms`);
      expect(duration).toBeLessThan(30000);
    });
  });
});

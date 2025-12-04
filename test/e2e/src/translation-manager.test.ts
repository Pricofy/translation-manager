import { LambdaClient, InvokeCommand } from '@aws-sdk/client-lambda';

const FUNCTION_NAME = 'pricofy-translation-manager';
const REGION = process.env.AWS_REGION || 'eu-west-1';

const lambda = new LambdaClient({ region: REGION });

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
  // Increase timeout for Lambda cold starts
  jest.setTimeout(30000);

  describe('Basic Translation', () => {
    it('should translate single text es→fr', async () => {
      const response = await invokeManager({
        texts: ['Hola mundo'],
        sourceLang: 'es',
        targetLang: 'fr',
      });

      expect(response.error).toBeUndefined();
      expect(response.translations).toHaveLength(1);
      expect(response.translations![0]).toBeTruthy();
      expect(response.chunksProcessed).toBe(1);
    });

    it('should translate batch of texts', async () => {
      const response = await invokeManager({
        texts: [
          'iPhone 12 Pro en buen estado',
          'MacBook Pro M2 poco uso',
          'Coche seminuevo con pocos kilómetros',
        ],
        sourceLang: 'es',
        targetLang: 'it',
      });

      expect(response.error).toBeUndefined();
      expect(response.translations).toHaveLength(3);
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

  describe('Language Pairs', () => {
    const pairs = [
      ['es', 'fr'],
      ['es', 'it'],
      ['es', 'pt'],
      ['es', 'de'],
      ['fr', 'es'],
      ['it', 'de'],
    ];

    test.each(pairs)('should translate %s→%s', async (source, target) => {
      const response = await invokeManager({
        texts: ['Prueba de traducción'],
        sourceLang: source,
        targetLang: target,
      });

      expect(response.error).toBeUndefined();
      expect(response.translations).toHaveLength(1);
    });
  });

  describe('Error Handling', () => {
    it('should return error for unsupported pair', async () => {
      const response = await invokeManager({
        texts: ['Hello'],
        sourceLang: 'es',
        targetLang: 'en', // English not supported
      });

      expect(response.error).toContain('no translator');
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
  });

  describe('Chunking', () => {
    it('should handle large batch with multiple chunks', async () => {
      // Generate 100 texts (~4000 tokens, should create 2+ chunks)
      const texts = Array.from(
        { length: 100 },
        (_, i) => `Producto número ${i} en venta con excelentes condiciones`
      );

      const response = await invokeManager({
        texts,
        sourceLang: 'es',
        targetLang: 'fr',
      });

      expect(response.error).toBeUndefined();
      expect(response.translations).toHaveLength(100);
      expect(response.chunksProcessed).toBeGreaterThanOrEqual(2);
    });

    it('should preserve text order across chunks', async () => {
      const texts = Array.from({ length: 50 }, (_, i) => `Texto número ${i}`);

      const response = await invokeManager({
        texts,
        sourceLang: 'es',
        targetLang: 'fr',
      });

      expect(response.error).toBeUndefined();
      expect(response.translations).toHaveLength(50);

      // Verify order by checking each translation contains corresponding number
      for (let i = 0; i < 50; i++) {
        expect(response.translations![i]).toContain(String(i));
      }
    });
  });

  describe('Performance', () => {
    it('should complete single text translation in < 5s', async () => {
      const start = Date.now();

      await invokeManager({
        texts: ['Prueba rápida'],
        sourceLang: 'es',
        targetLang: 'fr',
      });

      const duration = Date.now() - start;
      console.log(`Single text latency: ${duration}ms`);
      expect(duration).toBeLessThan(5000);
    });

    it('should complete batch translation in < 10s', async () => {
      const texts = Array.from({ length: 50 }, (_, i) => `Producto ${i}`);
      const start = Date.now();

      await invokeManager({
        texts,
        sourceLang: 'es',
        targetLang: 'it',
      });

      const duration = Date.now() - start;
      console.log(`Batch (50 texts) latency: ${duration}ms`);
      expect(duration).toBeLessThan(10000);
    });
  });
});

declare module 'vitest' {
  export function describe(name: string, fn: () => void): void;
  export function it(name: string, fn: () => void | Promise<void>): void;
  export const expect: any;
  export const vi: any;
  export function afterEach(fn: () => void): void;
}

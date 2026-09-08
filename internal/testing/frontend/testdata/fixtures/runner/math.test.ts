import { add } from './math';

test('add sums two numbers', () => {
  expect(add(1, 2)).toBe(3);
});

test('add handles zero', () => {
  expect(add(0, 4)).toBe(4);
});

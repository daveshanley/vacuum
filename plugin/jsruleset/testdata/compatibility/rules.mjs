import { createRulesetFunction } from '@stoplight/spectral-core';
import { oas3 } from '@stoplight/spectral-formats';

const check = createRulesetFunction({ input: null, options: null }, async (value, options, context) => {
  if (value !== 'Expected') return [{ message: 'Use the expected title.', path: context.path }];
});
export default {
  formats: [oas3],
  rules: {
    'title-required': {
      given: '$.info',
      severity: 'error',
      then: { field: 'title', function: check }
    }
  }
};

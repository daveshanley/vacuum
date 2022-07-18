import { html } from 'lit';
import { expect, fixture } from '@open-wc/testing';
import { RuleCategoryNavigationComponent } from './rule-category-navigation-component';

describe('RuleCategoryNavigationComponent', () => {
  let element: RuleCategoryNavigationComponent;

  before(async () => {
    element = await fixture<RuleCategoryNavigationComponent>(
      html`<rule-category-navigation>oh</rule-category-navigation>`
    );
  });

  it('renders nav', () => {
    const ul = element.shadowRoot!.querySelector('ul')!;
    expect(ul).to.exist;
  });
});

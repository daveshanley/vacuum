import { BaseComponent } from '../../ts/base-component';
import { html, css } from 'lit';
import { RuleSelectedEvent } from '../../model/events';
import { CategoryRuleComponent } from './category-rule-component';

export class CategoryRulesComponent extends BaseComponent {
  static get styles() {
    const rulesCss = css`
      ul.rule {
        margin: 0;
        padding: 0;
      }

      section {
        //max-height: 35vh;
        overflow-y: hidden;
      }
    `;

    return [rulesCss];
  }

  render() {
    return html`
      <section @ruleSelected=${this._ruleSelected}>
        <ul class="rule">
          <slot></slot>
        </ul>
      </section>
    `;
  }

  private _ruleSelected(evt: CustomEvent<RuleSelectedEvent>) {
    const rules = this.shadowRoot
      .querySelector('slot')
      .assignedElements({ flatten: true });

    rules.forEach((catRule: CategoryRuleComponent) => {
      if (catRule.ruleId != evt.detail.id) {
        catRule.otherRuleSelected();
      }
    });
  }
}

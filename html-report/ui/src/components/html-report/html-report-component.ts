import { BaseComponent } from '../../ts/base-component';
import { html } from 'lit';
import { CategoryActivatedEvent } from '../../model/events';
import { CategoryRuleComponent } from './category-rule-component';

export class HtmlReportComponent extends BaseComponent {
  render() {
    return html`
      <div @categoryActivated=${this._categoryActivatedListener}>
        <slot name="navigation"></slot>
        <slot name="reports"></slot>
      </div>
    `;
  }

  _categoryActivatedListener(e: CustomEvent<CategoryActivatedEvent>) {
    const categoryReports = document.querySelectorAll('category-report');
    const categoryRules = document.querySelectorAll('category-rule');
    const slot = this.shadowRoot
      .querySelector('slot')
      .assignedElements({ flatten: true });

    const description = slot[0]
      .querySelector('nav')
      .querySelector('#category-description');
    if (description) {
      description.innerHTML = e.detail.description;
    }

    categoryReports.forEach((element: HTMLElement) => {
      if (element.id == e.detail.id) {
        element.style.display = 'block';
      } else {
        element.style.display = 'none';
      }
    });

    categoryRules.forEach((rule: CategoryRuleComponent) => {
      rule.open = false;
    });
  }

  // _violationSelectedListener(e: CustomEvent<ViolationSelectedEvent>) {
  //   const slots = this.shadowRoot.querySelectorAll('slot');
  //   const drawer: ViolationDrawerComponent = slots[2].assignedElements({
  //     flatten: true,
  //   })[0] as ViolationDrawerComponent;
  //   drawer.ruleId = e.detail.id;
  //   drawer.message = e.detail.message;
  //   drawer.code = e.detail.renderedCode;
  //   drawer.show();
  // }
}

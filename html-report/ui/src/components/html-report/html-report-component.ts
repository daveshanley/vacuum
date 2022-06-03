import { BaseComponent } from '../../ts/base-component';
import { css, html } from 'lit';
import { CategoryActivatedEvent } from '../../model/events';
import { CategoryRuleComponent } from './category-rule-component';
import { ViolationDrawerComponent } from '../violation-drawer/violation-drawer-component';
import { ResultGridComponent } from './result-grid-component';
import { CategoryRulesComponent } from './category-rules-component';
import { CategoryReportComponent } from './category-report-component';

export class HtmlReportComponent extends BaseComponent {
  static get styles() {
    const report = css`
      .html-report {
        height: 100%;
      }
    `;

    return [report];
  }

  render() {
    return html`
      <div
        class="html-report"
        @categoryActivated=${this._categoryActivatedListener}
        @violationSelected=${this._violationSelectedListener}
      >
        <slot name="navigation"></slot>
        <slot name="reports"></slot>
      </div>
    `;
  }

  _categoryActivatedListener(e: CustomEvent<CategoryActivatedEvent>) {
    const categoryReports = document.querySelectorAll('category-report');
    const categoryRules = document.querySelectorAll('category-rule');
    const categoryRuleGroup = document.querySelectorAll('category-rules');
    const resultGrid = document.querySelector(
      'result-grid'
    ) as ResultGridComponent;

    const violationDrawer = document.querySelector(
      'violation-drawer'
    ) as ViolationDrawerComponent;

    const slot = this.shadowRoot
      .querySelector('slot')
      .assignedElements({ flatten: true });

    const description = slot[0]
      .querySelector('nav')
      .querySelector('#category-description');

    if (description) {
      description.innerHTML = e.detail.description;
    }

    categoryReports.forEach((element: CategoryReportComponent) => {
      if (element.id == e.detail.id) {
        element.style.display = 'block';
      } else {
        element.style.display = 'none';
      }
    });

    categoryRules.forEach((rule: CategoryRuleComponent) => {
      rule.open = false;
      //  console.log('do we hide?', rule.numResults)
    });

    categoryRuleGroup.forEach((rules: CategoryRulesComponent) => {
      if (rules.id == e.detail.id) {
        if (rules.rules && rules.rules.length <= 0) {
          rules.isEmpty = true;
        }
        //        console.log('this rule has ', rules.rules.length)
      }
    });

    if (violationDrawer) {
      violationDrawer.hide();
    }
    if (resultGrid) {
      //resultGrid.requestUpdate();
    }
  }

  _violationSelectedListener() {
    const violationDrawer = document.querySelector(
      'violation-drawer'
    ) as ViolationDrawerComponent;
    violationDrawer.show();
  }
}

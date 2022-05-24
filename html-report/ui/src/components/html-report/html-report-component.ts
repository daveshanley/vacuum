import { BaseComponent } from '../../ts/base-component';
import { html } from 'lit';
import {CategoryActivatedEvent, ViolationSelectedEvent} from '../../model/events';
import {ViolationDrawerComponent} from "../violation-drawer/violation-drawer-component";

export class HtmlReportComponent extends BaseComponent {
  render() {
    return html`
      <div @categoryActivated=${this._categoryActivatedListener}
           @violationSelected=${this._violationSelectedListener}>
        <slot name="navigation"></slot>
        <slot name="reports"></slot>
        <slot name="drawer"></slot>
      </div>
    `;
  }

  _categoryActivatedListener(e: CustomEvent<CategoryActivatedEvent>) {
    const elements = document.querySelectorAll('category-report');
    const slot = this.shadowRoot
      .querySelector('slot')
      .assignedElements({ flatten: true });

    const description = slot[0]
      .querySelector('nav')
      .querySelector('#category-description');
    if (description) {
      description.innerHTML = e.detail.description;
    }

    elements.forEach((element: HTMLElement) => {
      if (element.id == e.detail.id) {
        element.style.display = 'block';
      } else {
        element.style.display = 'none';
      }
    });
  }

  _violationSelectedListener(e: CustomEvent<ViolationSelectedEvent>) {
    const slots = this.shadowRoot.querySelectorAll('slot')
    const drawer: ViolationDrawerComponent = slots[2].assignedElements({flatten: true})[0] as ViolationDrawerComponent
    console.log('let us open the drawer!', drawer, e);
    drawer.show();
  }
}

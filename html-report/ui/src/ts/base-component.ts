import { LitElement } from 'lit';

export class BaseComponent extends LitElement {
  protected get _slottedChildren() {
    const slot = this.shadowRoot.querySelector('slot');
    if (slot) {
      return slot.assignedElements({ flatten: true });
    }
    return;
  }
}

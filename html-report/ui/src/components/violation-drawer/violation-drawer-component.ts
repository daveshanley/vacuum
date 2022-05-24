import {BaseComponent} from "../../ts/base-component";
import {html} from "lit";

export class ViolationDrawerComponent extends BaseComponent {
    render() {
        return html`
            <sl-drawer label="Drawer" placement="bottom" class="drawer-placement-bottom">
                This drawer slides in from the bottom.
                <sl-button slot="footer" variant="primary" @click=${this.hide}>Close</sl-button>
            </sl-drawer>
        `;
    }

    get drawer() {
        return this.shadowRoot.querySelector('sl-drawer')
    }

    public show() {
        this.drawer.show();
    }

    public hide() {
        this.drawer.hide();
    }
}
'use strict';

const Mustache = require('mustache');

module.exports = class Config {

    static get template_maestro() {
        return `
        network:
            interfaces:
                {{#interfaces}}
                {{content}}
                {{/interfaces}}
        devicedb_conn_config:
            devicedb_uri: \"https://{{device_id}}:9090\"
            devicedb_prefix: \"vagrant\"
            devicedb_bucket: \"lww\"
            relay_id: \"{{device_id}}\"
            ca_chain: \"$DEVICEDB_SRC/hack/certs/myCA.pem\"
        config_end: true`;
    }

    static get template_network_dhcp() {
        return `- if_name: {{interface_name}}
                  existing: override
                  dhcpv4: {{dhcp}}`;
    }

    static get template_network_static() {
        return `- if_name: {{interface_name}}
                  existing: override
                  dhcpv4: {{dhcp}}
                  ipv4_addr: {{ip_address}}
                  ipv4_mask: {{ip_mask}}`;
    }

    constructor() {
    }

    render_dhcp()
    {
        let template = '';
        if (this.dhcp === true) {
            template = Config.template_network_dhcp;
            return Mustache.render(template, this);
        }
        template = Config.template_network_static;
        return Mustache.render(template, this);
    }

    render(view)
    {
        if (!('interfaces' in view))
            return '';

        view.content = this.render_dhcp;

        let template = Config.template_maestro;
        return Mustache.render(template, view);
    }


}
'use strict';

var theme = {
    plain: {
        color: '#CCCAC2',
        backgroundColor: '#242936'
    },
    styles: [
        {
            types: ['comment', 'prolog', 'doctype', 'cdata'],
            style: {
                color: '#B8CFE680',
                fontStyle: 'italic'
            },
        },
        {
            types: ['namespace'],
            style: {
                opacity: 0.9
            },
        },
        {
            types: ['string', 'char', 'attr-value'],
            style: {
                color: '#D5FF80'
            },
        },
        {
            types: ['attribute'],
            style: {
                color: '#FFDFB3',
            },
        },
        {
            types: ['operator'],
            style: {
                color: '#F29E74'
            },
        },
        {
            types: ['entity', 'module-declaration', 'class-name', 'type-definition', 'url', 'symbol', 'variable', 'property'],
            style: {
                color: '#73D0FF'
            },
        },
        {
            types: ['regex'],
            style: {
                color: '#95E6CB',
            },
        },
        {
            types: ['constant', 'number', 'boolean'],
            style: {
                color: '#DFBFFF'
            },
        },
        {
            types: ['atrule', 'attr-name', 'selector'],
            style: {
                color: '#FA8D3E'
            },
        },
        {
            types: ['function', 'function-definition'],
            style: {
                color: '#FFD173'
            },
        },
        {
            types: ['function-variable'],
            style: {
                color: '#FFD173'
            },
        },
        {
            types: ['tag'],
            style: {
                color: '#5CCFE6',
            },
        },
        {
            types: ['selector', 'keyword'],
            style: {
                color: '#FFAD66'
            },
        },
        {
            types: ['inserted'],
            style: {
                color: '#87D96C',
            },
        },
        {
            types: ['deleted'],
            style: {
                color: '#F27983',
            },
        },
    ]
};

module.exports = theme;
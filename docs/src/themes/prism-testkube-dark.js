'use strict';

var theme = {
    plain: {
        color: '#BFBDB6',
        backgroundColor: '#0D1017'
    },
    styles: [
        {
            types: ['comment', 'prolog', 'doctype', 'cdata'],
            style: {
                color: '#ACB6BF8C',
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
                color: '#AAD94C'
            },
        },
        {
            types: ['attribute'],
            style: {
                color: '#E6B673',
            },
        },
        {
            types: ['operator'],
            style: {
                color: '#F29668'
            },
        },
        {
            types: ['entity', 'module-declaration', 'class-name', 'type-definition', 'url', 'symbol', 'variable', 'property'],
            style: {
                color: '#59C2FF'
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
                color: '#D2A6FF'
            },
        },
        {
            types: ['atrule', 'attr-name', 'selector'],
            style: {
                color: '#FF8F40'
            },
        },
        {
            types: ['function', 'function-definition'],
            style: {
                color: '#FFB454'
            },
        },
        {
            types: ['function-variable'],
            style: {
                color: '#FFB454'
            },
        },
        {
            types: ['tag'],
            style: {
                color: '#39BAE6',
            },
        },
        {
            types: ['selector', 'keyword'],
            style: {
                color: '#FF8F40'
            },
        },
        {
            types: ['inserted'],
            style: {
                color: '#7FD962',
            },
        },
        {
            types: ['deleted'],
            style: {
                color: '#F26D78',
            },
        },
    ]
};

module.exports = theme;
'use strict';

var theme = {
    plain: {
        color: '#5C6166',
        backgroundColor: '#FCFCFC'
    },
    styles: [
        {
            types: ['comment', 'prolog', 'doctype', 'cdata'],
            style: {
                color: '#787B8099',
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
                color: '#86B300'
            },
        },
        {
            types: ['attribute'],
            style: {
                color: '#E6BA7E',
            },
        },
        {
            types: ['tag'],
            style: {
                color: '#55B4D4',
            },
        },
        {
            types: ['operator'],
            style: {
                color: '#ED9366'
            },
        },
        {
            types: ['entity', 'module-declaration', 'class-name', 'type-definition', 'url', 'symbol', 'variable', 'property'],
            style: {
                color: '#399EE6'
            },
        },
        {
            types: ['regex'],
            style: {
                color: '#4CBF99',
            },
        },
        {
            types: ['constant', 'number', 'boolean'],
            style: {
                color: '#A37ACC'
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
                color: '#F2AE49'
            },
        },
        {
            types: ['function-variable'],
            style: {
                color: '#F2AE49'
            },
        },
        {
            types: ['selector', 'keyword'],
            style: {
                color: '#FA8D3E'
            },
        },
        {
            types: ['inserted'],
            style: {
                color: '#6CBF43',
            },
        },
        {
            types: ['deleted'],
            style: {
                color: '#FF7383',
            },
        },
    ]
};

module.exports = theme;
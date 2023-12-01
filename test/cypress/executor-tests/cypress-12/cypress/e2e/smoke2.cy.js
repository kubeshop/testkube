describe('Smoke test 2', () => {
  it(`expect 1=1`, () => {
    let value = '';
    for (let i = 0; i < 17000; i++) {
      value += 'a';
    }
    expect(value).to.equal('1');
  })
})

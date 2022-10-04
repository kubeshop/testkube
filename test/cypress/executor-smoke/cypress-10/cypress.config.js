module.exports = (on, config) => {
  on('task', {
    log(message) {
      console.log(message)

      return null
    },
  })

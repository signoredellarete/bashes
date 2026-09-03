export function createOrderedWriter(write) {
  let tail = Promise.resolve();

  return (data) => {
    const operation = tail.then(() => write(data));
    tail = operation.catch(() => {});
    return operation;
  };
}

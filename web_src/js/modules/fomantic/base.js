let ariaIdCounter = 0;

export function generateAriaId() {
  return `_aria_auto_id_${ariaIdCounter++}`;
}

export function linkLabelAndInput(label, input) {
  const labelFor = label.getAttribute('for');
  const inputId = input.getAttribute('id');

  if (inputId && !labelFor) { // missing "for"
    label.setAttribute('for', inputId);
  } else if (!inputId && !labelFor) { // missing both "id" and "for"
    const id = generateAriaId();
    input.setAttribute('id', id);
    label.setAttribute('for', id);
  }
}

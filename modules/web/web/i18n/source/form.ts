// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Form-related text in the English source locale.
 */
export default {
  // Validation messages.
  validation: {
    required: 'This field is required',
    email: 'Please enter a valid email address',
    url: 'Please enter a valid URL',
    number: 'Please enter a valid number',
    integer: 'Please enter a valid integer',
    minLength: 'Please enter at least {min} characters',
    maxLength: 'Please enter no more than {max} characters',
    minValue: 'Please enter a value greater than or equal to {min}',
    maxValue: 'Please enter a value less than or equal to {max}',
    pattern: 'Please enter a value matching the required pattern',
    passwordMatch: 'Passwords do not match',
  },

  // Placeholder text.
  placeholder: {
    search: 'Search...',
    select: 'Select an option',
    date: 'Select date',
    time: 'Select time',
    datetime: 'Select date and time',
    enterText: 'Enter text...',
  },

  // Form buttons.
  button: {
    submit: 'Submit',
    reset: 'Reset',
    cancel: 'Cancel',
    search: 'Search',
    filter: 'Filter',
  },

  // Common form labels.
  label: {
    username: 'Username',
    password: 'Password',
    confirmPassword: 'Confirm Password',
    email: 'Email',
    phone: 'Phone',
    address: 'Address',
    dateOfBirth: 'Date of Birth',
    gender: 'Gender',
    firstName: 'First Name',
    lastName: 'Last Name',
  },
} as const;

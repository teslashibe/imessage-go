package imessage

import "encoding/json"

// parseJXAResult unmarshals JXA stdout JSON into the given type.
func parseJXAResult[T any](raw string) (*T, error) {
	var v T
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return nil, err
	}
	return &v, nil
}

const jsxListContacts = `
(function() {
  var Contacts = Application('Contacts');
  var people = Contacts.people();
  var query = (INPUT.query || '').toLowerCase();
  var limit = INPUT.limit || 50;
  var offset = INPUT.offset || 0;

  var filtered = [];
  for (var i = 0; i < people.length; i++) {
    var p = people[i];
    var fn = p.firstName() || '';
    var ln = p.lastName() || '';
    var co = p.organization() || '';
    if (query) {
      var full = (fn + ' ' + ln + ' ' + co).toLowerCase();
      if (full.indexOf(query) === -1) continue;
    }
    filtered.push(p);
  }

  var total = filtered.length;
  var slice = filtered.slice(offset, offset + limit);
  var results = [];
  for (var j = 0; j < slice.length; j++) {
    var pp = slice[j];
    var phones = [];
    try {
      var phs = pp.phones.value();
      for (var k = 0; k < phs.length; k++) phones.push(phs[k]);
    } catch(e) {}
    var emails = [];
    try {
      var ems = pp.emails.value();
      for (var k = 0; k < ems.length; k++) emails.push(ems[k]);
    } catch(e) {}
    results.push({
      id: pp.id(),
      firstName: pp.firstName() || '',
      lastName: pp.lastName() || '',
      company: pp.organization() || '',
      phones: phones,
      emails: emails
    });
  }
  return JSON.stringify({ contacts: results, total: total });
})();
`

const jsxCreateContact = `
(function() {
  var Contacts = Application('Contacts');
  var p = Contacts.Person({
    firstName: INPUT.first_name || '',
    lastName: INPUT.last_name || '',
    organization: INPUT.company || ''
  });
  // JXA requires push() into the people collection — Contacts.add(p)
  // returns "No error. (0)" silently and never persists.
  Contacts.people.push(p);

  var phones = INPUT.phones || [];
  for (var i = 0; i < phones.length; i++) {
    p.phones.push(Contacts.Phone({ value: phones[i], label: 'mobile' }));
  }
  var emails = INPUT.emails || [];
  for (var i = 0; i < emails.length; i++) {
    p.emails.push(Contacts.Email({ value: emails[i], label: 'home' }));
  }
  Contacts.save();
  return JSON.stringify({
    ok: true,
    id: p.id(),
    firstName: p.firstName() || '',
    lastName: p.lastName() || ''
  });
})();
`

const jsxUpdateContact = `
(function() {
  var Contacts = Application('Contacts');
  var people = Contacts.people.whose({ id: INPUT.id });
  if (people.length === 0) throw new Error('contact not found: ' + INPUT.id);
  var p = people[0];

  if (INPUT.first_name !== null && INPUT.first_name !== undefined) {
    p.firstName = INPUT.first_name;
  }
  if (INPUT.last_name !== null && INPUT.last_name !== undefined) {
    p.lastName = INPUT.last_name;
  }
  if (INPUT.company !== null && INPUT.company !== undefined) {
    p.organization = INPUT.company;
  }
  var addPhones = INPUT.add_phones || [];
  for (var i = 0; i < addPhones.length; i++) {
    p.phones.push(Contacts.Phone({ value: addPhones[i], label: 'mobile' }));
  }
  var addEmails = INPUT.add_emails || [];
  for (var i = 0; i < addEmails.length; i++) {
    p.emails.push(Contacts.Email({ value: addEmails[i], label: 'home' }));
  }
  Contacts.save();
  return JSON.stringify({
    ok: true,
    id: p.id(),
    firstName: p.firstName() || '',
    lastName: p.lastName() || ''
  });
})();
`

const jsxDeleteContact = `
(function() {
  var Contacts = Application('Contacts');
  var people = Contacts.people.whose({ id: INPUT.id });
  if (people.length === 0) throw new Error('contact not found: ' + INPUT.id);
  Contacts.delete(people[0]);
  Contacts.save();
  return JSON.stringify({ ok: true, id: INPUT.id });
})();
`
